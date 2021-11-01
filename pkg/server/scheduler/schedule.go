package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/server/constants"
	"github.com/Congrool/nodes-grouping/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
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

func (e *extender) Prioritize(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error) {
	// TODO:
	return nil, nil
}

func (e *extender) Filter(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error) {
	args := extenderArgs.DeepCopy()
	pod := args.Pod
	nodes := args.Nodes
	policyNamespaceName, ok := pod.Annotations[constants.GroupingScheduleKey]
	if !ok {
		// the pod doesn't need grouping schedule
		// return as it is.
		return &extenderv1.ExtenderFilterResult{
			Nodes:     args.Nodes,
			NodeNames: args.NodeNames,
		}, nil
	}

	policyNs, policyName, err := utils.ParseNamespaceName(policyNamespaceName)
	if err != nil {
		return nil, err
	}

	// get policy in pod's annotation
	policy := &policyv1alpha1.PropagationPolicy{}
	if err := e.client.Get(e.ctx, client.ObjectKey{Namespace: policyNs, Name: policyName}, policy); err != nil {
		return nil, err
	}

	// get all target clusterNames
	var clusterNames []string
	for _, targetWeight := range policy.Spec.Placement.StaticWeightList {
		clusterNames = append(clusterNames, targetWeight.TargetCluster.ClusterNames...)
	}
	clusters, err := utils.GetClustersWithName(e.ctx, e.client, clusterNames)
	if err != nil {
		klog.Errorf("failed to get clusters obj according to their names, err: %v", err)
		return nil, err
	}

	// get map that map node to cluster it belongs to
	nodesInClusters, err := utils.GetNodesInClusters(e.ctx, e.client, clusters)
	if err != nil {
		klog.Errorf("failed to get nodes in clusters when filter the nodes, %v", err)
		return nil, err
	}

	// filter nodes that are not in target clusters
	filterdNodes := []corev1.Node{}
	filterdNodeNames := []string{}
	for _, node := range nodes.Items {
		if _, ok := nodesInClusters[node.Name]; ok {
			filterdNodes = append(filterdNodes, node)
			filterdNodeNames = append(filterdNodeNames, node.Name)
		}
	}
	filterdNodeList := args.Nodes.DeepCopy()
	filterdNodeList.Items = filterdNodes
	return &extenderv1.ExtenderFilterResult{
		Nodes:     filterdNodeList,
		NodeNames: &filterdNodeNames,
	}, nil
}

func NewSchedulerExtender(ctx context.Context, client client.Client) SchedulerExtender {
	return &extender{
		ctx:    ctx,
		client: client,
	}
}

func WithFilterHandler(filterFunc func(*extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var extenderArgs *extenderv1.ExtenderArgs
		var extenderResults *extenderv1.ExtenderFilterResult

		defer func() {
			w.Header().Set("Content-Type", "application/json")
			responseBody, err := json.Marshal(extenderResults)
			if err != nil {
				klog.Errorf("failed to marshal filter extenderResults, %v", err)
				return
			}
			w.Write(responseBody)
		}()

		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)

		// decode
		if err := json.NewDecoder(body).Decode(extenderArgs); err != nil {
			klog.Errorf("failed to decode filter extenderArgs, request: %s, err: %v", buf.String(), err)
			extenderResults = &extenderv1.ExtenderFilterResult{
				Error: err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("get request info: %v", buf.String())

		// do filter
		extenderResults, err := filterFunc(extenderArgs)
		if err != nil {
			klog.Errorf("failed to run filter handler, err: %v", err)
			extenderResults = &extenderv1.ExtenderFilterResult{
				Error: err.Error(),
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}
