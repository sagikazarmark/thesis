package awsactivities

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
)

type AutoScaling struct {
	Client *autoscaling.Client
}

func (a AutoScaling) DetachInstances(ctx context.Context, params *autoscaling.DetachInstancesInput) (*autoscaling.DetachInstancesOutput, error) {
	return a.Client.DetachInstances(ctx, params)
}
