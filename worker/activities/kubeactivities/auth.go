package kubeactivities

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go/aws/session"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

type KubeClientFactory struct {
	EKSClient      *eks.Client
	TokenGenerator token.Generator
	Session        *session.Session
}

func (f *KubeClientFactory) NewClientset(ctx context.Context, clusterName string) (*kubernetes.Clientset, error) {
	describeClusterOutput, err := f.EKSClient.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(clusterName)})
	if err != nil {
		return nil, err
	}

	tok, err := f.TokenGenerator.GetWithOptions(&token.GetTokenOptions{
		ClusterID: clusterName,
		Session:   f.Session,
	})
	if err != nil {
		return nil, err
	}

	ca, err := base64.StdEncoding.DecodeString(aws.ToString(describeClusterOutput.Cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(
		&rest.Config{
			Host:        aws.ToString(describeClusterOutput.Cluster.Endpoint),
			BearerToken: tok.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: ca,
			},
		},
	)
}

type ClusterSetup struct {
	KubeClientFactory KubeClientFactory
}

type CreateAuthConfigMapInput struct {
	ClusterName          string
	NodeInstanceRoleARNs []string
}

func (s ClusterSetup) CreateAuthConfigMap(ctx context.Context, input CreateAuthConfigMapInput) error {
	clientset, err := s.KubeClientFactory.NewClientset(ctx, input.ClusterName)
	if err != nil {
		return err
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-auth",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"mapRoles": s.createAuthConfigMapRoles(input.NodeInstanceRoleARNs),
		},
	}

	_, err = clientset.CoreV1().ConfigMaps("kube-system").Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (s ClusterSetup) createAuthConfigMapRoles(nodeInstanceRoleARNs []string) string {
	var mapRoles string

	for _, nodeInstanceRoleARN := range nodeInstanceRoleARNs {
		mapRoles = fmt.Sprintf("%s  - rolearn: %s\n    username: system:node:{{EC2PrivateDNSName}}\n    groups:\n      - system:bootstrappers\n      - system:nodes\n", mapRoles, nodeInstanceRoleARN)
	}

	return mapRoles
}
