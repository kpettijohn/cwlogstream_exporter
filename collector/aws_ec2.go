package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"github.com/kpettijohn/cwlogstream_exporter/types"
)

// AWSEC2Client contains an AWS EC2 ec2iface client
type AWSEC2Client struct {
	client ec2iface.EC2API
}

// AWSEC2InstanceGatherer interface implments methods to gather AWS EC2 instance data
type AWSEC2InstanceGatherer interface {
	GetInstances(context.Context, string) (*types.AWSEC2DescribeInstances, error)
}

// NewAWSEC2Client creates a new AWS EC2 API client
func NewAWSEC2Client(awsRegion string) (*AWSEC2Client, error) {
	// Create AWS session
	s, err := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	if err != nil {
		return nil, fmt.Errorf("Error creating AWS session")
	}

	return &AWSEC2Client{
		client: ec2.New(s),
	}, nil
}

// GetInstances fetches all running instances using AWSEC2DescribeInstances
func (c *AWSEC2Client) GetInstances(ctx context.Context, tagFilter string) (*types.AWSEC2DescribeInstances, error) {
	var err error
	filters := []*ec2.Filter{
		{
			Name: aws.String("instance-state-name"),
			Values: []*string{
				aws.String("running"),
			},
		},
	}
	splitTags := strings.Split(tagFilter, ",")
	for _, tagString := range splitTags {
		tag := strings.Split(tagString, ":")
		tagName, tagValue := tag[0], tag[1]
		f := &ec2.Filter{
			Name: aws.String(fmt.Sprintf("tag:%s", tagName)),
			Values: []*string{
				aws.String(tagValue),
			},
		}
		filters = append(filters, f)
	}
	instances := &types.AWSEC2DescribeInstances{}
	params := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	// Get EC2 instances
	err = c.client.DescribeInstancesPagesWithContext(ctx, params,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, r := range page.Reservations {
				for _, i := range r.Instances {
					instances.Instances = append(instances.Instances, i)
				}
			}

			return true
		})

	if err != nil {
		return nil, err
	}

	return instances, nil
}
