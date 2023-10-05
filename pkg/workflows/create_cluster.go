package workflows

import (
	"go.temporal.io/sdk/workflow"
)

// CreateClusterInput contains the input parameters for the {ClusterCreate} workflow.
type CreateClusterInput struct{}

// CreateClusterOutput contains the return parameters for the {ClusterCreate} workflow.
type CreateClusterOutput struct{}

// CreateCluster creates a new EKS cluster.
func CreateCluster(ctx workflow.Context, input CreateClusterInput) (*CreateClusterOutput, error) {
	// Create VPC (using cloudformation)
	// Wait for cloudformation stack
	// Create cluster
	// Wait for cluster
	// Create self-managed node group (using cloudformation)
	// Wait for cloudformation stack
	return nil, nil
}
