package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	extenderutil "github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/utils"
	"github.com/Congrool/nodes-grouping/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Filter interface {
	Filter(*extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error)
}

type FilterPlugin interface {
	Name() string
	FilterNodes(context.Context, client.Client, *corev1.Pod, []corev1.Node, *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error)
}

type filter struct {
	ctx           context.Context
	client        client.Client
	filterPlugins []FilterPlugin
	// TODO:
	// func to get policy
}

func (f *filter) Filter(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error) {
	if extenderArgs == nil {
		return nil, fmt.Errorf("empty extenderArgs")
	}

	args := extenderArgs.DeepCopy()
	pod := args.Pod

	_, policy, err := utils.GetRelativeDeployAndPolicy(f.ctx, f.client, pod)
	if err != nil {
		klog.Errorf("failed to get relative policy for pod %s/%s, %v", pod.Namespace, pod.Name, err)
		return f.constructFilterResult(args.Nodes.Items), err
	}

	if policy == nil {
		klog.V(2).Infof("can not find policy for pod %s/%s, skip filter", pod.Namespace, pod.Name)
		return f.constructFilterResult(args.Nodes.Items), nil
	}

	var nodes []corev1.Node
	var errs []error
	nodes = append(nodes, args.Nodes.Items...)
	for _, filterPlugin := range f.filterPlugins {
		var err error
		nodes, err = filterPlugin.FilterNodes(f.ctx, f.client, pod, nodes, policy)
		if err != nil {
			klog.Errorf("failed to filter nodes for pod %s/%s according to policy %s/%s with plugin %s, %v",
				pod.Namespace, pod.Name,
				policy.Namespace, policy.Name,
				filterPlugin.Name(), err)
			errs = append(errs, err)
		}
		nodeNames := make([]string, len(nodes))
		for _, node := range nodes {
			nodeNames = append(nodeNames, node.Name)
		}
		klog.V(2).Infof("after filter plugin: %s, nodes: %v ", filterPlugin.Name(), nodeNames)
	}
	return f.constructFilterResult(nodes), errors.NewAggregate(errs)
}

func (f *filter) constructFilterResult(nodes []corev1.Node) *extenderv1.ExtenderFilterResult {
	if nodes == nil {
		return &extenderv1.ExtenderFilterResult{}
	}
	filteredNodeNames := make([]string, len(nodes))
	for i := range nodes {
		filteredNodeNames[i] = nodes[i].Name
	}

	nodeList := &corev1.NodeList{}
	nodeList.Items = append(nodeList.Items, nodes...)

	filterResults := &extenderv1.ExtenderFilterResult{
		Nodes:     nodeList,
		NodeNames: &filteredNodeNames,
	}
	return filterResults
}

func New(ctx context.Context, client client.Client) Filter {
	return &filter{
		ctx:    ctx,
		client: client,
		filterPlugins: []FilterPlugin{
			&enoughPodsFilter{},
			&notInNodeGroupsFilter{},
		},
	}
}

func WithFilterHandler(filterFunc func(*extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var extenderResults *extenderv1.ExtenderFilterResult
		defer func() {
			w.Header().Set("Content-Type", "application/json")
			responseBody, err := json.Marshal(extenderResults)
			if err != nil {
				klog.Errorf("failed to marshal filter extenderResults, %v", err)
				responseBody = nil
			}
			w.Write(responseBody)
		}()

		extenderArgs, err := extenderutil.ExtractExtenderArgsFromRequest(r)
		if err != nil {
			extenderResults = &extenderv1.ExtenderFilterResult{
				Error: err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// do filter
		extenderResults, err = filterFunc(extenderArgs)
		if err != nil {
			klog.Errorf("failed to run filter handler, err: %v", err)
			extenderResults = &extenderv1.ExtenderFilterResult{
				Error: err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})
}
