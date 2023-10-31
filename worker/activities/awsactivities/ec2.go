package awsactivities

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go/middleware"
	"go.temporal.io/sdk/activity"
)

type EC2 struct {
	Client *ec2.Client
}

func (e EC2) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return e.Client.TerminateInstances(ctx, params)
}

func (e EC2) WaitForInstanceTerminated(ctx context.Context, instanceIds []string) error {
	info := activity.GetInfo(ctx)

	waiter := ec2.NewInstanceTerminatedWaiter(e.Client, func(o *ec2.InstanceTerminatedWaiterOptions) {
		if info.HeartbeatTimeout > 0 {
			// Set the max delay to something less than the heartbeat timeout (if there is any)
			//
			// For example: if heartbeat is 100 seconds, max delay is 95 seconds (5 second to make sure the heartbeat gets to Temporal in time).
			// If heartbeat is 10 seconds, max delay is 8 seconds (giving heartbeat 2 seconds to arrive).
			//
			// Note: 5 seconds is just an arbitrary number.
			o.MaxDelay = info.HeartbeatTimeout - min(time.Duration(float64(info.HeartbeatTimeout)*0.2), 5*time.Second)
		}

		// There is heartbeat, so there is a max delay. We need to make sure the min delay is smaller than that.
		// Default min delay is 30 seconds.
		//
		// For example: if max delay is 95 seconds, min delay is 30 seconds.
		// If max delay is 8 seconds, min delay is 1.6 seconds (20% of max delay).
		if o.MaxDelay > 0 {
			o.MinDelay = min(time.Duration(float64(info.HeartbeatTimeout)*0.2), 30*time.Second)
		}

		o.APIOptions = append(o.APIOptions, func(s *middleware.Stack) error {
			return s.Initialize.Add(heartbeat{}, middleware.Before)
		})
	})

	params := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}

	return waiter.Wait(ctx, params, info.Deadline.Sub(info.StartedTime))
}
