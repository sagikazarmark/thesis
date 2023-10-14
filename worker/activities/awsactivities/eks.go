package awsactivities

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/smithy-go/middleware"
	"go.temporal.io/sdk/activity"
)

type EKS struct {
	Client *eks.Client
}

func (e EKS) CreateCluster(ctx context.Context, params *eks.CreateClusterInput) (*eks.CreateClusterOutput, error) {
	if params.ClientRequestToken == nil {
		info := activity.GetInfo(ctx)

		params.ClientRequestToken = aws.String(fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, info.ActivityID))
	}

	return e.Client.CreateCluster(ctx, params)
}

func (e EKS) WaitForClusterActive(ctx context.Context, clusterName string) error {
	info := activity.GetInfo(ctx)

	waiter := eks.NewClusterActiveWaiter(e.Client, func(o *eks.ClusterActiveWaiterOptions) {
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

	params := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	return waiter.Wait(ctx, params, info.Deadline.Sub(info.StartedTime))
}

func (e EKS) DeleteCluster(ctx context.Context, clusterName string) (*eks.DeleteClusterOutput, error) {
	params := &eks.DeleteClusterInput{
		Name: aws.String(clusterName),
	}

	return e.Client.DeleteCluster(ctx, params)
}

func (e EKS) WaitForClusterDeleted(ctx context.Context, clusterName string) error {
	info := activity.GetInfo(ctx)

	waiter := eks.NewClusterDeletedWaiter(e.Client, func(o *eks.ClusterDeletedWaiterOptions) {
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

	params := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	return waiter.Wait(ctx, params, info.Deadline.Sub(info.StartedTime))
}
