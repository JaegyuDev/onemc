package aws

import (
    "context"
    "fmt"
    cfg "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/ec2/types"
    "log"
    "time"
)

var (
    ec2Client *ec2.Client
)

func init() {
    defaultConfig, err := cfg.LoadDefaultConfig(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    ec2Client = ec2.NewFromConfig(defaultConfig)
}

func StartInstanceByID(id string) {
    fmt.Printf("Starting instance %s...\n", id)
    instances, err := ec2Client.StartInstances(context.Background(), &ec2.StartInstancesInput{
        InstanceIds: []string{id},
    })
    if err != nil {
        log.Fatal(err)
    }

    var state types.InstanceStateName
    for state != "running" {
        time.Sleep(5 * time.Second)
        instances, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
            InstanceIds: []string{id},
        })
        if err != nil {
            log.Fatal(err)
        }

        state = instances.Reservations[0].Instances[0].State.Name
    }

    fmt.Printf("Instance %q has started\n", *instances.StartingInstances[0].InstanceId)
}

func StopInstanceByID(id string) {
    fmt.Printf("Stopping instance %s...\n", id)

    instances, err := ec2Client.StopInstances(context.Background(), &ec2.StopInstancesInput{
        InstanceIds: []string{id},
    })
    if err != nil {
        log.Fatal(err)
    }

    var state types.InstanceStateName
    for state != "stopped" {
        time.Sleep(5 * time.Second)
        instances, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
            InstanceIds: []string{id},
        })
        if err != nil {
            log.Fatal(err)
        }

        state = instances.Reservations[0].Instances[0].State.Name
    }

    fmt.Printf("Instance %q has started\n", *instances.StoppingInstances[0].InstanceId)
}
