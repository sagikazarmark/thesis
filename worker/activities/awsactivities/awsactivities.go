package awsactivities

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/smithy-go/middleware"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

// RegisterActivities registers AWS related activities in a Temporal Worker.
func RegisterActivities(w worker.Worker) {
	// CloudFormation
	{

		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			panic(err)
		}
		client := cloudformation.NewFromConfig(cfg)

		a := CloudFormation{
			Client: client,
		}

		w.RegisterActivity(a.CreateStack)
		w.RegisterActivity(a.WaitForCreateStack)

		w.RegisterActivity(a.DeleteStack)
		w.RegisterActivity(a.WaitForDeleteStack)

		w.RegisterActivity(a.DescribeStacks)
	}

	// EKS
	{

		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			panic(err)
		}
		client := eks.NewFromConfig(cfg)

		a := EKS{
			Client: client,
		}

		w.RegisterActivity(a.CreateCluster)
		w.RegisterActivity(a.WaitForClusterActive)

		w.RegisterActivity(a.DeleteCluster)
		w.RegisterActivity(a.WaitForClusterDeleted)
	}
}

type heartbeat struct{}

func (heartbeat) ID() string {
	return "TemporalHeartbeat"
}

func (heartbeat) HandleInitialize(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
	out middleware.InitializeOutput, metadata middleware.Metadata, err error,
) {
	activity.RecordHeartbeat(ctx)

	return next.HandleInitialize(ctx, in)
}
