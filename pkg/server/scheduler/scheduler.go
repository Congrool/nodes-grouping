package scheduler

import (
	"context"

	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SchedulerExtender interface {
	Filter(*extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error)
	Prioritize(*extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error)
	// Bind function is not supported
}

type extender struct {
	client client.Client
	ctx    context.Context
}

func NewSchedulerExtender(ctx context.Context, client client.Client) SchedulerExtender {
	return &extender{
		ctx:    ctx,
		client: client,
	}
}
