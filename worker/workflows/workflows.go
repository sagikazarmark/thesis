package workflows

import "go.temporal.io/sdk/worker"

func RegisterWorkflows(w worker.Worker) {
	w.RegisterWorkflow(CreateCluster)
	w.RegisterWorkflow(DeleteCluster)
	w.RegisterWorkflow(UpdateNodeGroup)
}
