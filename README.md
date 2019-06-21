# AWS CloudWatch log stream exporter

AWS CloudWatch log stream exporter (`cwlogstream_exporter`) collects metrics on AWS CloudWatch log stream last even times.

## Requirements
  - Each instance should create its own AWS log stream
    - Instance ID should be included in the log stream name
  - All instances within an AWS region should send logs to a common log group
    - Example: `/var/log/messages`
  - IAM permissions:
    - `ec2:DescribeInstances`
    - `logs:DescribeLogGroups`
    - `logs:DescribeLogStreams`

## Workflow

1. Fetch all log groups associated to a log group prefix.
2. Fetch all running EC2 instances.
3. For each log group, collect all log group streams
4. For each AWS instance check to see if a log group stream exists
5. Check log group last event time for the instance in question, if no events within the last hour consider the log stream dead.

## Usage

Basic usage

```
cwlogstream_exporter -aws.region=us-east-1 -aws.log-group-prefix=/var/log/messages -debug
```

## Build

```
go build
```
