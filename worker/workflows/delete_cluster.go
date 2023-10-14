package workflows

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/sagikazarmark/thesis/worker/activities/awsactivities"
	"go.temporal.io/sdk/workflow"

	"github.com/sagikazarmark/thesis/worker/cluster"
)

// DeleteClusterInput contains the input parameters for the [DeleteCluster] workflow.
type DeleteClusterInput struct {
	Cluster cluster.Cluster
}

// DeleteClusterOutput contains the return parameters for the [DeleteCluster] workflow.
type DeleteClusterOutput struct{}

// DeleteCluster creates a new EKS cluster.
func DeleteCluster(ctx workflow.Context, input DeleteClusterInput) (*DeleteClusterOutput, error) {
	if err := input.Cluster.Validate(); err != nil {
		return nil, err
	}

	var cfactivities awsactivities.CloudFormation
	var eksactivities awsactivities.EKS

	for _, ng := range input.Cluster.NodeGroups {
		ngStackName := fmt.Sprintf("%s-%s", input.Cluster.Name, ng.Name)

		// Delete self-managed node group (using cloudformation)
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 15 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			input := &cloudformation.DeleteStackInput{
				StackName: aws.String(ngStackName),
			}

			err := workflow.ExecuteActivity(ctx, cfactivities.DeleteStack, input).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// Wait for cloudformation stack
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 10 * time.Minute,
				HeartbeatTimeout:    30 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			err := workflow.ExecuteActivity(ctx, cfactivities.WaitForDeleteStack, ngStackName).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	// Delete cluster
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		err := workflow.ExecuteActivity(ctx, eksactivities.DeleteCluster, input.Cluster.Name).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	// Wait for cluster
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		err := workflow.ExecuteActivity(ctx, eksactivities.WaitForClusterDeleted, input.Cluster.Name).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	vpcStackName := fmt.Sprintf("%s-vpc", input.Cluster.Name)

	// Delete VPC (using cloudformation)
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := &cloudformation.DeleteStackInput{
			StackName: aws.String(vpcStackName),
		}

		err := workflow.ExecuteActivity(ctx, cfactivities.DeleteStack, input).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	// Wait for cloudformation stack
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		err := workflow.ExecuteActivity(ctx, cfactivities.WaitForDeleteStack, vpcStackName).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
