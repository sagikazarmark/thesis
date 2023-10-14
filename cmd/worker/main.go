package main

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"

	"github.com/sagikazarmark/thesis/worker/activities/awsactivities"
	"github.com/sagikazarmark/thesis/worker/activities/kubeactivities"
	"github.com/sagikazarmark/thesis/worker/workflows"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("starting worker")

	temporalClient, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
		Logger:   log.NewStructuredLogger(logger.With(slog.String("subsystem", "temporal"))),
	})
	if err != nil {
		logger.Error("unable to create Temporal Client", slog.Any("error", err))
	}
	defer temporalClient.Close()

	w := worker.New(temporalClient, "thesis", worker.Options{})

	workflows.RegisterWorkflows(w)
	awsactivities.RegisterActivities(w)
	kubeactivities.RegisterActivities(w)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("unable to start Worker", slog.Any("error", err))

		os.Exit(1)
	}
}
