# AWS CloudWatch log stream exporter

AWS CloudWatch log stream exporter (`cwlogstream_exporter`) collects metrics on AWS CloudWatch log stream last even times for EC2 instances.

## Requirements
  - Each instance creates its own AWS log stream
    - Instance ID should be included in the log stream name
  - All instances within an AWS region send logs to a common log group
    - Example: `/var/log/messages`
  - IAM permissions:
    - `ec2:DescribeInstances`
    - `logs:DescribeLogGroups`
    - `logs:DescribeLogStreams`

## Workflow

1. Fetch log group.
2. Fetch all running EC2 instances.
3. Collect all log group streams for the given log group.
4. For each AWS instance check to see if a log group stream exists
5. Check log group last event time for the instance in question, if no events within the last hour consider the log stream dead.

## Usage

Basic usage

```
cwlogstream_exporter -aws.region=us-east-1 -aws.log-group-name=/var/log/messages -aws.ec2-tag-filter=Env:dev,Compliance:level1
```

By default, metrics are exposed at `:9520/metrics`.
```
# HELP cwlogstream_sending Checks if AWS CloudWatch logs are being sent by AWS instance ID and log group
# TYPE cwlogstream_sending gauge
cwlogstream_sending{group="/var/log/messages",instance_id="i-0abc123def456ghkl"} 1
cwlogstream_sending{group="/var/log/messages",instance_id="i-0123abc456def7899"} 1
# HELP cwlogstream_up Checks if the exporter is up/online.
# TYPE cwlogstream_up gauge
cwlogstream_up 1
```

Example Prometheus job:

```
  - job_name: 'cwlogstream-exporter'
    scrape_interval: 5m
    scrape_timeout: 30s
    ec2_sd_configs:
      - region: us-east-1
        port: 9520
        role_arn: arn:aws:iam::012345678901:role/monitoring
    relabel_configs:
      - source_labels: [__meta_ec2_tag_CWLogStream_Exporter]
        regex: true
        action: keep
```

Example Prometheus alert:
```
- name: cwlogstream
  rules:
  - alert: CloudWatchLogStreamLastEvent
    expr: cwlogstream_sending == 0
    for: 30m
    annotations:
      summary: 'An AWS EC2 instance is not sending logs to AWS CloudWatch'
      description: 'Instance {{$labels.instance_id}} is not sending logs to the {{$labels.group}} CloudWatch log group'
```
## Build

```
go build
```

## Acknowledgments

Exporter was based on the following Prometheus exporters:
  - https://github.com/houserater/awslogs-exporter (fork of ecs-exporter)
  - https://github.com/slok/ecs-exporter
