package types

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// AWSLogGroupStreams contains a CloudWatch log group along with a list of its log streams
type AWSLogGroupStreams struct {
	Group   *AWSLogGroup                // Log group reference
	Streams []*cloudwatchlogs.LogStream // Log streams within the log group
}

// AWSLogGroup represents a CloudWatch log group ARN and name
type AWSLogGroup struct {
	ID   string // ARN of the log group
	Name string // Name of the log group
}

// AWSEC2DescribeInstances contains a list of AWS EC2 instances
type AWSEC2DescribeInstances struct {
	Instances []*ec2.Instance
}
