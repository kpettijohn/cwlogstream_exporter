package collector

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"

	"github.com/kpettijohn/cwlogstream_exporter/types"
)

// AWSLogsGatherer interface implments methods to gather data on a AWS CloudWatch log group and its log streams
type AWSLogsGatherer interface {
	GetLogGroups() ([]*types.AWSLogGroup, error)
	GetLogStreams(group *types.AWSLogGroup) (*types.AWSLogGroupStreams, error)
}

// AWSLogsClient contains an AWS CloudWatch Logs cloudwatchlogsiface client
type AWSLogsClient struct {
	client             cloudwatchlogsiface.CloudWatchLogsAPI
	logGroupNamePrefix *string
	logHistory         int64
}

// NewAWSLogsClient creates a new AWS CloudWatch Logs API client
func NewAWSLogsClient(awsRegion string, logGroupNamePrefix *string) (*AWSLogsClient, error) {
	// Create AWS session
	s := session.New(&aws.Config{Region: aws.String(awsRegion)})
	if s == nil {
		return nil, fmt.Errorf("error creating aws session")
	}

	return &AWSLogsClient{
		client:             cloudwatchlogs.New(s),
		logGroupNamePrefix: logGroupNamePrefix,
	}, nil
}

// GetLogGroups returns all log groups under a perfix
func (c *AWSLogsClient) GetLogGroups() ([]*types.AWSLogGroup, error) {
	params := &cloudwatchlogs.DescribeLogGroupsInput{}
	if len(*c.logGroupNamePrefix) > 0 {
		params.LogGroupNamePrefix = c.logGroupNamePrefix
	}

	// Get log groups
	resp, err := c.client.DescribeLogGroups(params)
	if err != nil {
		return nil, err
	}

	lg := []*types.AWSLogGroup{}
	for _, l := range resp.LogGroups {
		respLogGroup := &types.AWSLogGroup{
			ID:   aws.StringValue(l.Arn),
			Name: aws.StringValue(l.LogGroupName),
		}
		lg = append(lg, respLogGroup)
	}

	return lg, nil
}

// GetLogStreams will return all log streams for a group
func (c *AWSLogsClient) GetLogStreams(group *types.AWSLogGroup) (*types.AWSLogGroupStreams, error) {
	var err error
	logStreams := &types.AWSLogGroupStreams{
		Group: group,
	}
	desc := true
	orderBy := "LastEventTime"
	limit := int64(50)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		Descending:   &desc,
		LogGroupName: &group.Name,
		OrderBy:      &orderBy,
		Limit:        &limit,
	}

	/* Only iterate over 3 pages (150 log streams). Streams are sorted by last event time,
	   so unless you are running more than 150 instances it should include all of the
	   active log streams. */
	count := 0
	err = c.client.DescribeLogStreamsPages(params,
		func(page *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
			for _, s := range page.LogStreams {
				logStreams.Streams = append(logStreams.Streams, s)
			}
			count++
			if count > 3 {
				return false
			}
			return true
		})

	if err != nil {
		return nil, err
	}

	return logStreams, nil
}
