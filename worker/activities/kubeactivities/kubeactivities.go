package kubeactivities

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.temporal.io/sdk/worker"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

// RegisterActivities registers AWS related activities in a Temporal Worker.
func RegisterActivities(w worker.Worker) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	client := eks.NewFromConfig(cfg)

	tokenGenerator, _ := token.NewGenerator(true, false)

	sess := session.Must(session.NewSession())

	kubeClientFactory := KubeClientFactory{
		EKSClient:      client,
		TokenGenerator: tokenGenerator,
		Session:        sess,
	}

	// Cluster setup
	{

		a := ClusterSetup{
			KubeClientFactory: kubeClientFactory,
		}

		w.RegisterActivity(a.CreateAuthConfigMap)
	}

	// Nodes
	{

		a := Nodes{
			KubeClientFactory: kubeClientFactory,
		}

		w.RegisterActivity(a.ListNodes)
		w.RegisterActivity(a.DeleteNode)
		w.RegisterActivity(a.DrainNode)
	}
}
