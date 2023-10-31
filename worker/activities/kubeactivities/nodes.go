package kubeactivities

import (
	"context"
	"io"
	"time"

	"go.temporal.io/sdk/activity"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/drain"
)

type Nodes struct {
	KubeClientFactory KubeClientFactory
}

type ListNodesInput struct {
	ClusterName string
}

type ListNodesOutput struct {
	Nodes []v1.Node
}

func (n Nodes) ListNodes(ctx context.Context, input ListNodesInput) (*ListNodesOutput, error) {
	clientset, err := n.KubeClientFactory.NewClientset(ctx, input.ClusterName)
	if err != nil {
		return nil, err
	}

	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return &ListNodesOutput{
		Nodes: nodeList.Items,
	}, nil
}

type DeleteNodeInput struct {
	ClusterName string
	NodeName    string
}

type DeleteNodeOutput struct{}

func (n Nodes) DeleteNode(ctx context.Context, input DeleteNodeInput) (*DeleteNodeOutput, error) {
	clientset, err := n.KubeClientFactory.NewClientset(ctx, input.ClusterName)
	if err != nil {
		return nil, err
	}

	err = clientset.CoreV1().Nodes().Delete(ctx, input.NodeName, metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &DeleteNodeOutput{}, nil
}

type DrainNodeInput struct {
	ClusterName string
	NodeName    string
}

type DrainNodeOutput struct{}

func (n Nodes) DrainNode(ctx context.Context, input DrainNodeInput) (*DrainNodeOutput, error) {
	clientset, err := n.KubeClientFactory.NewClientset(ctx, input.ClusterName)
	if err != nil {
		return nil, err
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, input.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cordonHelper := drain.NewCordonHelper(node)

	if cordonHelper.UpdateIfRequired(true) {
		err, patchErr := cordonHelper.PatchOrReplace(clientset, false)
		if patchErr != nil {
			return nil, patchErr
		}

		if err != nil {
			return nil, err
		}
	}

	drainHelper := &drain.Helper{
		Client:              clientset,
		Force:               true,
		GracePeriodSeconds:  0,
		IgnoreAllDaemonSets: true,
		DisableEviction:     true, // This is generally a bad idea, but it's fine here
		Timeout:             10 * time.Second,
		DeleteEmptyDirData:  true,
		Selector:            "",
		PodSelector:         "",
		Out:                 io.Discard,
		ErrOut:              io.Discard,
		OnPodDeletedOrEvicted: func(pod *v1.Pod, usingEviction bool) {
			activity.RecordHeartbeat(ctx)
		},
	}

	err = drain.RunNodeDrain(drainHelper, input.NodeName)
	if err != nil {
		return nil, err
	}

	return &DrainNodeOutput{}, nil
}
