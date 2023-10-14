package awsactivities

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/smithy-go/middleware"
	"go.temporal.io/sdk/activity"
)

type CloudFormation struct {
	Client *cloudformation.Client
}

func (cf CloudFormation) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error) {
	if params.ClientRequestToken == nil {
		info := activity.GetInfo(ctx)

		params.ClientRequestToken = aws.String(fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, info.ActivityID))
	}

	// TODO: retryable errors
	return cf.Client.CreateStack(ctx, params)
}

func (cf CloudFormation) WaitForCreateStack(ctx context.Context, stackName string) error {
	info := activity.GetInfo(ctx)

	waiter := cloudformation.NewStackCreateCompleteWaiter(cf.Client, func(o *cloudformation.StackCreateCompleteWaiterOptions) {
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

	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}

	return waiter.Wait(ctx, params, info.Deadline.Sub(info.StartedTime))
}

func (cf CloudFormation) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	if params.ClientRequestToken == nil {
		info := activity.GetInfo(ctx)

		params.ClientRequestToken = aws.String(fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, info.ActivityID))
	}

	// TODO: retryable errors
	return cf.Client.DeleteStack(ctx, params)
}

func (cf CloudFormation) WaitForDeleteStack(ctx context.Context, stackName string) error {
	info := activity.GetInfo(ctx)

	waiter := cloudformation.NewStackDeleteCompleteWaiter(cf.Client, func(o *cloudformation.StackDeleteCompleteWaiterOptions) {
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

	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}

	return waiter.Wait(ctx, params, info.Deadline.Sub(info.StartedTime))
}

func (cf CloudFormation) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	// TODO: retryable errors
	return cf.Client.DescribeStacks(ctx, params)
}
