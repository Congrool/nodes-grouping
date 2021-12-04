package extender

import (
	"context"

	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/filter"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/prioritizer"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SchedulerExtender interface {
	Filter(*extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error)
	Prioritize(*extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error)
	// Bind function is not supported
}

type extender struct {
	client      client.Client
	ctx         context.Context
	prioritizer prioritizer.Prioritizer
	filter      filter.Filter
}

func (e *extender) Filter(args *extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error) {
	return e.filter.Filter(args)
}

func (e *extender) Prioritize(args *extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error) {
	return e.prioritizer.Prioritize(args)
}

func NewSchedulerExtender(ctx context.Context, client client.Client) SchedulerExtender {
	return &extender{
		ctx:         ctx,
		client:      client,
		prioritizer: prioritizer.New(ctx, client),
		filter:      filter.New(ctx, client),
	}
}
