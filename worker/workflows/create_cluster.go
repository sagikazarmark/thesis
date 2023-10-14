package workflows

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/sagikazarmark/thesis/worker/activities/awsactivities"
	"github.com/sagikazarmark/thesis/worker/activities/kubeactivities"
	"go.temporal.io/sdk/workflow"

	"github.com/sagikazarmark/thesis/worker/cftemplates"
	"github.com/sagikazarmark/thesis/worker/cluster"
)

// CreateClusterInput contains the input parameters for the [CreateCluster] workflow.
type CreateClusterInput struct {
	Cluster cluster.Cluster
}

// CreateClusterOutput contains the return parameters for the [CreateCluster] workflow.
type CreateClusterOutput struct{}

// CreateCluster creates a new EKS cluster.
func CreateCluster(ctx workflow.Context, input CreateClusterInput) (*CreateClusterOutput, error) {
	if err := input.Cluster.Validate(); err != nil {
		return nil, err
	}

	var cfactivities awsactivities.CloudFormation
	var eksactivities awsactivities.EKS
	var clusterSetupActivities kubeactivities.ClusterSetup

	vpcStackName := fmt.Sprintf("%s-vpc", input.Cluster.Name)

	// Create VPC (using cloudformation)
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := &cloudformation.CreateStackInput{
			StackName:    aws.String(vpcStackName),
			TemplateBody: aws.String(cftemplates.VPC()),
		}

		err := workflow.ExecuteActivity(ctx, cfactivities.CreateStack, input).Get(ctx, nil)
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

		err := workflow.ExecuteActivity(ctx, cfactivities.WaitForCreateStack, vpcStackName).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	// Grab VPC details
	var vpcID, subnetIDs, securityGroupIDs string
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := &cloudformation.DescribeStacksInput{
			StackName: aws.String(vpcStackName),
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
			case "VpcId":
				vpcID = aws.ToString(output.OutputValue)
			case "SubnetIds":
				subnetIDs = aws.ToString(output.OutputValue)
			case "SecurityGroups":
				securityGroupIDs = aws.ToString(output.OutputValue)
			}
		}
	}

	workflow.GetLogger(ctx).Info("VPC details", "VPCID", vpcID, "SubnetIDs", subnetIDs, "securityGroupIDs", securityGroupIDs)

	// Create cluster
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := &eks.CreateClusterInput{
			Name: aws.String(input.Cluster.Name),
			ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
				SecurityGroupIds: strings.Split(securityGroupIDs, ","),
				SubnetIds:        strings.Split(subnetIDs, ","),
			},
			RoleArn: aws.String(input.Cluster.Cloud.RoleARN),
			Version: aws.String(input.Cluster.Kubernetes.Version),
		}

		err := workflow.ExecuteActivity(ctx, eksactivities.CreateCluster, input).Get(ctx, nil)
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

		err := workflow.ExecuteActivity(ctx, eksactivities.WaitForClusterActive, input.Cluster.Name).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	var nodeInstanceRoleARNs []string

	for _, ng := range input.Cluster.NodeGroups {
		ngStackName := fmt.Sprintf("%s-%s", input.Cluster.Name, ng.Name)

		// Create self-managed node group (using cloudformation)
		{
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 15 * time.Second,
			}
			ctx := workflow.WithActivityOptions(ctx, ao)

			stackParameters := []cftypes.Parameter{
				{
					ParameterKey:   aws.String("ClusterName"),
					ParameterValue: aws.String(input.Cluster.Name),
				},
				{
					ParameterKey:   aws.String("NodeGroupName"),
					ParameterValue: aws.String(fmt.Sprintf("%s-%s", input.Cluster.Name, ng.Name)),
				},
				{
					ParameterKey:   aws.String("VpcId"),
					ParameterValue: aws.String(vpcID),
				},
				{
					ParameterKey:   aws.String("Subnets"),
					ParameterValue: aws.String(subnetIDs),
				},
				{
					ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
					ParameterValue: aws.String(securityGroupIDs),
				},
			}

			if ng.KeyName != "" {
				stackParameters = append(stackParameters, cftypes.Parameter{
					ParameterKey:   aws.String("KeyName"),
					ParameterValue: aws.String(ng.KeyName),
				})
			}

			input := &cloudformation.CreateStackInput{
				StackName:    aws.String(ngStackName),
				TemplateBody: aws.String(cftemplates.NodeGroup()),
				Capabilities: []cftypes.Capability{
					cftypes.CapabilityCapabilityIam,
				},
				Parameters: stackParameters,
			}

			err := workflow.ExecuteActivity(ctx, cfactivities.CreateStack, input).Get(ctx, nil)
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

			err := workflow.ExecuteActivity(ctx, cfactivities.WaitForCreateStack, ngStackName).Get(ctx, nil)
			if err != nil {
				return nil, err
			}
		}

		// Grab node group details
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
				case "NodeInstanceRole":
					nodeInstanceRoleARNs = append(nodeInstanceRoleARNs, aws.ToString(output.OutputValue))
				}
			}
		}
	}

	// Create auth ConfigMap
	{
		ao := workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
		}
		ctx := workflow.WithActivityOptions(ctx, ao)

		input := kubeactivities.CreateAuthConfigMapInput{
			ClusterName:          input.Cluster.Name,
			NodeInstanceRoleARNs: nodeInstanceRoleARNs,
		}

		err := workflow.ExecuteActivity(ctx, clusterSetupActivities.CreateAuthConfigMap, input).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
