package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// default CLI flag values
const (
	defaultListenAddress        = ":9520"
	defaultAwsRegion            = ""
	defaultMetricsPath          = "/metrics"
	defaultDebug                = false
	defaultLogGroupPrefix       = ""
	defaultlogStreamTimeout     = time.Duration(60 * time.Minute)
	defaultEC2InstanceTagFilter = "Env:dev"
)

// cfg is the global exporter configuration
var cfg *config

// parse global config
func parse(args []string) error {
	return cfg.parse(args)
}

// config contains CLI flags
type config struct {
	fs *flag.FlagSet

	listenAddress        string
	awsRegion            string
	metricsPath          string
	debug                bool
	logGroupPrefix       string
	logStreamTimeout     time.Duration
	ec2InstanceTagFilter string
}

// init will load all the flags
func init() {
	cfg = new()
}

// new returns an initialized config
func new() *config {
	c := &config{
		fs: flag.NewFlagSet(os.Args[0], flag.ContinueOnError),
	}

	c.fs.StringVar(
		&c.listenAddress, "web.listen-address", defaultListenAddress, "Address to listen on")

	c.fs.StringVar(
		&c.awsRegion, "aws.region", defaultAwsRegion, "The AWS region to get metrics from")

	c.fs.StringVar(
		&c.metricsPath, "web.telemetry-path", defaultMetricsPath, "The path where metrics will be exposed")

	c.fs.StringVar(
		&c.logGroupPrefix, "aws.log-group-prefix", defaultLogGroupPrefix, "AWS logs group prefix")

	c.fs.DurationVar(
		&c.logStreamTimeout, "aws.log-stream-timeout", defaultlogStreamTimeout, "Timeout for when to consider an AWS log stream dead")

	c.fs.StringVar(
		&c.ec2InstanceTagFilter, "aws.ec2-tag-filter", defaultEC2InstanceTagFilter, "AWS EC2 tag filter")

	c.fs.BoolVar(
		&c.debug, "debug", defaultDebug, "Run exporter in debug mode")

	return c
}

// parse parses flags for configuration
func (c *config) parse(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}

	if len(c.fs.Args()) != 0 {
		return fmt.Errorf("Invalid command line arguments. Help: %s -h", os.Args[0])
	}

	if c.awsRegion == "" {
		return fmt.Errorf("An aws region is required")
	}

	return nil
}
