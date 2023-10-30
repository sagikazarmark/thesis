package cluster

import (
	"errors"
	"fmt"
)

// Cluster describes the desired state of a cluster.
type Cluster struct {
	Name       string
	Cloud      Cloud
	Kubernetes ClusterKubernetes

	NodeGroups []NodeGroup
}

func (c Cluster) Validate() error {
	if c.Name == "" {
		return errors.New("cluster name is required")
	}

	if err := c.Cloud.Validate(); err != nil {
		return fmt.Errorf("cloud: %w", err)
	}

	if err := c.Kubernetes.Validate(); err != nil {
		return fmt.Errorf("kubernetes: %w", err)
	}

	for _, ng := range c.NodeGroups {
		if err := ng.Validate(); err != nil {
			return fmt.Errorf("node group(%s): %w", ng.Name, err)
		}
	}

	return nil
}

type Cloud struct {
	RoleARN string
}

func (c Cloud) Validate() error {
	if c.RoleARN == "" {
		return errors.New("role ARN is required")
	}

	return nil
}

type ClusterKubernetes struct {
	Version string
}

func (k ClusterKubernetes) Validate() error {
	if k.Version == "" {
		return errors.New("version is required")
	}

	return nil
}

type NodeGroup struct {
	Name       string
	KeyName    string
	Kubernetes NodeGroupKubernetes
}

func (ng NodeGroup) Validate() error {
	if ng.Name == "" {
		return errors.New("name is required")
	}

	if ng.KeyName == "" {
		return errors.New("key name is required")
	}

	if err := ng.Kubernetes.Validate(); err != nil {
		return fmt.Errorf("kubernetes: %w", err)
	}

	return nil
}

type NodeGroupKubernetes struct {
	Version string
}

func (k NodeGroupKubernetes) Validate() error {
	if k.Version == "" {
		return errors.New("version is required")
	}

	return nil
}
