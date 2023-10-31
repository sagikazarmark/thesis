package workflows

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/sagikazarmark/thesis/worker/activities/awsactivities"
	"github.com/sagikazarmark/thesis/worker/activities/kubeactivities"
	"github.com/sagikazarmark/thesis/worker/cftemplates"
	"go.temporal.io/sdk/workflow"
	v1 "k8s.io/api/core/v1"
)

// UpdateClusterInput contains the input parameters for the [UpdateCluster] workflow.
type UpdateClusterInput struct {
	ClusterName       string
	NodeGroupName     string
	KubernetesVersion string
}

func (i UpdateClusterInput) Validate() error {
	if i.ClusterName == "" {
		return errors.New("cluster name is required")
	}

	if i.NodeGroupName == "" {
		return errors.New("node group name is required")
	}

	if i.KubernetesVersion == "" {
		return errors.New("kubernetes version is required")
	}

	return nil
}

// UpdateClusterOutput contains the return parameters for the [UpdateCluster] workflow.
type UpdateClusterOutput struct{}

// UpdateCluster creates a new EKS cluster.
func UpdateCluster(ctx workflow.Context, input UpdateClusterInput) (*UpdateClusterOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var cfactivities awsactivities.CloudFormation
	var nodeactivities kubeactivities.Nodes
	var asgactivities awsactivities.AutoScaling
	var ec2activities awsactivities.EC2

	ngStackName := fmt.Sprintf("%s-%s", input.ClusterName, input.NodeGroupName)

	// Update self-managed node group (using cloudformation)
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		stackParameters := []cftypes.Parameter{
			{
				ParameterKey:     aws.String("ClusterName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("NodeGroupName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("VpcId"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("Subnets"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("ClusterControlPlaneSecurityGroup"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:     aws.String("KeyName"),
				UsePreviousValue: aws.Bool(true),
			},
			{
				ParameterKey:   aws.String("NodeImageIdSSMParam"),
				ParameterValue: aws.String(fmt.Sprintf("/aws/service/eks/optimized-ami/%s/amazon-linux-2/recommended/image_id", input.KubernetesVersion)),
			},
		}

		input := &cloudformation.UpdateStackInput{
			StackName:    aws.String(ngStackName),
			TemplateBody: aws.String(cftemplates.NodeGroup()),
			Capabilities: []cftypes.Capability{
				cftypes.CapabilityCapabilityIam,
			},
			Parameters: stackParameters,
		}

		// TODO: handle no update error
		err := workflow.ExecuteActivity(ctx, cfactivities.UpdateStack, input).Get(ctx, nil)
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

		err := workflow.ExecuteActivity(ctx, cfactivities.WaitForUpdateStack, ngStackName).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	// Grab node group details
	var asgName string
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := &cloudformation.DescribeStacksInput{
			StackName: aws.String(ngStackName),
		}

		var output *cloudformation.DescribeStacksOutput

		err := workflow.ExecuteActivity(ctx, cfactivities.DescribeStacks, input).Get(ctx, &output)
		if err != nil {
			return nil, err
		}

		if len(output.Stacks) == 0 {
			return nil, errors.New("stack not found")
		}

		for _, output := range output.Stacks[0].Outputs {
			switch aws.ToString(output.OutputKey) {
			case "NodeAutoScalingGroup":
				asgName = aws.ToString(output.OutputValue)
			}
		}
	}

	workflow.GetLogger(ctx).Info("node group details", "asg", asgName)

	var nodes []v1.Node
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := kubeactivities.ListNodesInput{
			ClusterName: input.ClusterName,
		}

		var output kubeactivities.ListNodesOutput

		err := workflow.ExecuteActivity(ctx, nodeactivities.ListNodes, input).Get(ctx, &output)
		if err != nil {
			return nil, err
		}

		nodes = output.Nodes
	}

	for _, node := range nodes {
		providerIDSegments := strings.Split(strings.TrimPrefix(node.Spec.ProviderID, "aws:///"), "/")
		region, instanceID := providerIDSegments[0], providerIDSegments[1]

		workflow.GetLogger(ctx).Info("node details", "name", node.Name, "providerID", node.Spec.ProviderID, "region", region, "instanceId", instanceID)

		// Drain node
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			input := kubeactivities.DrainNodeInput{
				ClusterName: input.ClusterName,
				NodeName:    node.Name,
			}

			err := workflow.ExecuteActivity(ctx, nodeactivities.DrainNode, input).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// Delete node
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 15 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			input := kubeactivities.DeleteNodeInput{
				ClusterName: input.ClusterName,
				NodeName:    node.Name,
			}

			err := workflow.ExecuteActivity(ctx, nodeactivities.DeleteNode, input).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// TODO: verify node is gone

		// Detach instance from ASG
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			input := &autoscaling.DetachInstancesInput{
				InstanceIds:                    []string{instanceID},
				AutoScalingGroupName:           aws.String(asgName),
				ShouldDecrementDesiredCapacity: aws.Bool(false),
			}

			err := workflow.ExecuteActivity(ctx, asgactivities.DetachInstances, input).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// Terminate instance
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 15 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			input := &ec2.TerminateInstancesInput{
				InstanceIds: []string{instanceID},
			}

			err := workflow.ExecuteActivity(ctx, ec2activities.TerminateInstances, input).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// Wait for instance termination
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 5 * time.Minute,
				HeartbeatTimeout:    30 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			err := workflow.ExecuteActivity(ctx, ec2activities.WaitForInstanceTerminated, []string{instanceID}).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// TODO: wait for node to rejoin
		workflow.Sleep(ctx, time.Minute)
	}

	return nil, nil
}
