package collector

import (
	"context"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/kpettijohn/cwlogstream_exporter/internal/log"
	"github.com/kpettijohn/cwlogstream_exporter/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Prometheus exporter namespace
	namespace = "cwlogstream"
	// Timeout when making AWS API calls to collect AWS CloudWatch log group streams
	timeout = 10 * time.Second
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Checks if the exporter is up/online. ",
		nil,
		nil,
	)

	awsLogsSending = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "sending"),
		"Checks if AWS CloudWatch logs are being sent by AWS instance ID and log group",
		[]string{"instance_id", "group"},
		nil,
	)
)

// Exporter collects AWS Logs metrics
type Exporter struct {
	sync.Mutex                                 // Our exporter object will be locakble to protect from concurrent scrapes
	client              AWSLogsGatherer        // Custom AWS Logs client to get information from the log groups
	ec2Client           AWSEC2InstanceGatherer // Custom AWS EC2 client to get information from the log groups
	region              string                 // The region where the exporter will scrape
	timeout             time.Duration          // The timeout for the whole gathering process
	instanceIDRegexp    *regexp.Regexp         // AWS instance ID regexp
	lastLogEventTimeout time.Time              // Timeout for when to consdier a log stream dead
	wg                  *sync.WaitGroup        // Collector WaitGroup
	ec2TagFilter        string
}

// New returns an initialized exporter
func New(awsRegion string, logGroupPrefix string, logStreamTimeout time.Duration, ec2TagFilter string) (*Exporter, error) {
	timeNow := time.Now().UTC()
	wg := sync.WaitGroup{}
	lastEventTimeCutOff := timeNow.Add((-logStreamTimeout))
	logClient, err := NewAWSLogsClient(awsRegion, &logGroupPrefix)
	if err != nil {
		return nil, err
	}

	ec2Client, err := NewAWSEC2Client(awsRegion)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		Mutex:               sync.Mutex{},
		client:              logClient,
		ec2Client:           ec2Client,
		region:              awsRegion,
		timeout:             timeout,
		instanceIDRegexp:    regexp.MustCompile(`i-([a-z0-9]{8,17})`),
		lastLogEventTimeout: lastEventTimeCutOff,
		wg:                  &wg,
		ec2TagFilter:        ec2TagFilter,
	}, nil
}

// sendSafeMetric uses context to cancel the send over a closed channel.
// If a main function finishes (for example due to to timeout), the goroutines running in background will
// try to send metrics over a closed channel, this will panic, this way the context will check first
// if the iteraiton has been finished and dont let continue sending the metric
func sendSafeMetric(ctx context.Context, ch chan<- prometheus.Metric, metric prometheus.Metric) error {
	// Check if iteration has finished
	select {
	case <-ctx.Done():
		log.Errorf("Tried to send a metric after collection context has finished, metric: %s", metric)
		return ctx.Err()
	default: // continue
	}
	// If no then send the metric
	ch <- metric
	return nil
}

// Describe describes all the metrics ever exported by the exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- awsLogsSending
}

// Collect fetches the stats from configured AWS Logs and delivers them
// as Prometheus metrics. It implements prometheus.Collector
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e.Lock()
	defer e.Unlock()

	// Get log groups
	logGroups, err := e.client.GetLogGroups()
	if err != nil {
		sendSafeMetric(ctx, ch, prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0))
		log.Errorf("Error collecting metrics: %v", err)
		return
	}

	// Get EC2 instances
	instances, err := e.ec2Client.GetInstances(e.ec2TagFilter)
	if err != nil {
		sendSafeMetric(ctx, ch, prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0))
		log.Errorf("Error collecting metrics: %v", err)
		return
	}

	// Start getting metrics per log group
	errCh := make(chan bool)
	waitCh := make(chan struct{})
	collecterCount := len(logGroups)
	e.wg.Add(collecterCount)
	for _, lg := range logGroups {
		go func(lg types.AWSLogGroup) {
			// Get all log streams for a group
			lgs, err := e.client.GetLogStreams(&lg)
			if err != nil {
				errCh <- true
				log.Errorf("Error collecting log group stream metrics: %v", err)
				return
			}
			err = e.collectLogGroupStreamMetrics(ctx, ch, lgs, instances)
			if err != nil {
				errCh <- true
				log.Errorf("Error collecting log stream metrics: %v", err)
				return
			}
			errCh <- false
		}(*lg)
	}

	go func() {
		e.wg.Wait()
		close(waitCh)
	}()

	result := float64(1)

	for {
		select {
		case <-waitCh:
			collecterCount--
			log.Debugf("A sync.WaitGroup finished. Current count: %v", collecterCount)
		case err := <-errCh:
			if err {
				result = 0
			}
		case <-time.After(e.timeout):
			log.Errorf("Error collecting metrics: Timeout making calls, waited for %v  without response", e.timeout)
			result = 0
			break

		default:
			if collecterCount <= 0 || len(logGroups) == 0 {
				break
			}
		}
		if collecterCount <= 0 || len(logGroups) == 0 || result == 0 {
			break
		}
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, result,
	)
}

func (e *Exporter) collectLogGroupStreamMetrics(ctx context.Context, ch chan<- prometheus.Metric, lg *types.AWSLogGroupStreams, instances *types.AWSEC2DescribeInstances) error {
	// Close our sync.WaitGroup after processing all instances for a log group
	defer e.wg.Done()
	// Iterate over all instances from a EC2 DescribeInstances API call
	for _, instance := range instances.Instances {
		if e.containsInstanceLogStream(lg, *instance.InstanceId) {
			log.Debugf("Found a log stream for instance_id=%s within the log group.\n", *instance.InstanceId)
			stream := e.lookupLogStreamByInstance(lg, *instance.InstanceId)
			instanceID := e.instanceIDRegexp.FindString(*stream.LogStreamName)
			if *instance.InstanceId == instanceID {
				// LastEventTimestamp - number of milliseconds after Jan 1, 1970 00:00:00 UTC.
				// https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_LogStream.html#CWL-Type-LogStream-lastEventTimestamp
				lastEventTimestamp := time.Unix(0, *stream.LastEventTimestamp*int64(time.Millisecond)).UTC()
				log.Debugf("Last event log int64 in milliseconds: %v", *stream.LastEventTimestamp)
				log.Debugf("Last event log timestamp: %v", lastEventTimestamp)
				log.Debugf("Last event configured cutoff timestamp: %v", e.lastLogEventTimeout)
				// When the log stream's last event time is before the exporter's configured last event timeout,
				// set the metric to down (0), otherwise consider the log stream up (1).
				if lastEventTimestamp.Before(e.lastLogEventTimeout) {
					log.Debugf("awsLogsSending (found): metric=0 instance_id=%s log_group_name=%s\n", instanceID, lg.Group.Name)
					err := sendSafeMetric(ctx, ch, prometheus.MustNewConstMetric(awsLogsSending, prometheus.GaugeValue, 0, instanceID, lg.Group.Name))
					if err != nil {
						return err
					}
				} else {
					log.Debugf("awsLogsSending (not found): metric=1 instance_id=%s log_group_name=%s\n", *instance.InstanceId, lg.Group.Name)
					err := sendSafeMetric(ctx, ch, prometheus.MustNewConstMetric(awsLogsSending, prometheus.GaugeValue, 1, *instance.InstanceId, lg.Group.Name))
					if err != nil {
						return err
					}
				}
			}
		} else {
			// If no matching log stream exists for a instance consider it down (0)
			log.Debugf("awsLogsSending (not found): metric=0 instance_id=%s log_group_name=%s\n", *instance.InstanceId, lg.Group.Name)
			err := sendSafeMetric(ctx, ch, prometheus.MustNewConstMetric(awsLogsSending, prometheus.GaugeValue, 0, *instance.InstanceId, lg.Group.Name))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Check if a log group stream name contains a given instance ID
func (e *Exporter) containsInstanceLogStream(streams *types.AWSLogGroupStreams, instance string) bool {
	log.Debugf("Streams: %v", streams)
	for _, s := range streams.Streams {
		instanceID := e.instanceIDRegexp.FindString(*s.LogStreamName)
		log.Debugf("Log stream name: %s, Instance regex: %s", *s.LogStreamName, instanceID)
		if instanceID == instance {
			return true
		}
	}
	log.Debugf("No log stream found with instance_id in name: %s", instance)
	return false
}

// Lookup a log group stream by instance
func (e *Exporter) lookupLogStreamByInstance(streams *types.AWSLogGroupStreams, instance string) *cloudwatchlogs.LogStream {
	logStreams := []cloudwatchlogs.LogStream{}
	for _, s := range streams.Streams {
		instanceID := e.instanceIDRegexp.FindString(*s.LogStreamName)

		if instanceID == instance {
			logStreams = append(logStreams, *s)
		}
	}
	log.Debugf("Lookup log stream by instance_id: %v", logStreams)
	sort.Slice(logStreams, func(i, j int) bool {
		return *logStreams[i].LastEventTimestamp > *logStreams[j].LastEventTimestamp
	})
	return &logStreams[0]
}
