package prioritizer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	extenderutil "github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/utils"
	"github.com/Congrool/nodes-grouping/pkg/utils"
)

type Prioritizer interface {
	Prioritize(*extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error)
}

type PrioritizerPlugin interface {
	Name() string
	PrioritizeNodes(context.Context, client.Client, *corev1.Pod, *extenderv1.ExtenderArgs, *groupmanagementv1alpha1.PropagationPolicy) (extenderv1.HostPriorityList, error)
}

type prioritizer struct {
	ctx                context.Context
	client             client.Client
	prioritizerPlugins []PrioritizerPlugin
}

func (p *prioritizer) Prioritize(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error) {
	if extenderArgs == nil {
		return nil, fmt.Errorf("empty extenderArgs")
	}

	args := extenderArgs.DeepCopy()
	pod := args.Pod

	_, policy, err := utils.GetRelativeDeployAndPolicy(p.ctx, p.client, pod)
	if err != nil {
		klog.Errorf("failed to get relative policy for pod %s/%s, %v", pod.Namespace, pod.Name, err)
		return p.notScore(args)
	}
	if policy == nil {
		klog.V(2).Infof("can not find policy for pod %s/%s, skip prioritize", pod.Namespace, pod.Name)
		return p.notScore(args)
	}

	var priorityList extenderv1.HostPriorityList
	var errs []error

	for _, plugins := range p.prioritizerPlugins {
		var err error
		scores, err := plugins.PrioritizeNodes(p.ctx, p.client, pod, extenderArgs, policy)
		if err != nil {
			klog.Errorf("failed to score node according to policy %s/%s when scheduling pod %s/%s, %v",
				policy.Namespace, policy.Name,
				pod.Namespace, pod.Name,
				err)
			errs = append(errs, err)
			continue
		}
		priorityList = p.combineScores(priorityList, scores)
	}

	return &priorityList, errors.NewAggregate(errs)
}

func (p *prioritizer) notScore(args *extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error) {
	nodeList := args.Nodes
	if nodeList == nil {
		return &extenderv1.HostPriorityList{}, nil
	}

	nodesNum := len(nodeList.Items)
	prioritList := make([]extenderv1.HostPriority, nodesNum)
	for i := range nodeList.Items {
		prioritList = append(prioritList, extenderv1.HostPriority{
			Host:  nodeList.Items[i].Name,
			Score: 0,
		})
	}
	return (*extenderv1.HostPriorityList)(&prioritList), nil
}

func (p *prioritizer) combineScores(old extenderv1.HostPriorityList, new extenderv1.HostPriorityList) extenderv1.HostPriorityList {
	if len(old) == 0 {
		// init
		return new
	}

	newscores := make(map[string]int64, len(new))
	for i := range new {
		newscores[new[i].Host] = new[i].Score
	}
	for i := range old {
		old[i].Score += newscores[old[i].Host]
	}
	return old
}

func New(ctx context.Context, client client.Client) Prioritizer {
	return &prioritizer{
		ctx:    ctx,
		client: client,
		prioritizerPlugins: []PrioritizerPlugin{
			&diffBasedPrioritizePlugin{},
		},
	}
}

func WithPrioritizeHander(prioritizeFunc func(*extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var extenderArgs *extenderv1.ExtenderArgs
		var prioritizeHostList *extenderv1.HostPriorityList

		defer func() {
			w.Header().Set("Content-Type", "application/json")
			responseBody, err := json.Marshal(prioritizeHostList)
			if err != nil {
				klog.Errorf("failed to marshal prioritize hostList, %v", err)
				responseBody = nil
			}
			w.Write(responseBody)
		}()

		extenderArgs, err := extenderutil.ExtractExtenderArgsFromRequest(r)
		// decode
		if err != nil {
			prioritizeHostList = nil
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// do prioritize
		prioritizeHostList, err = prioritizeFunc(extenderArgs)
		if err != nil {
			klog.Errorf("failed to run prioritize handler, err: %v", err)
			prioritizeHostList = nil
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

	})
}
