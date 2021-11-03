package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (e *extender) Filter(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.ExtenderFilterResult, error) {
	args := extenderArgs.DeepCopy()
	pod := args.Pod
	policyNamespaceName, ok := pod.Annotations[constants.GroupingScheduleKey]
	returnUnmodified := func() (*extenderv1.ExtenderFilterResult, error) {
		return &extenderv1.ExtenderFilterResult{
			Nodes:     args.Nodes,
			NodeNames: args.NodeNames,
		}, nil
	}
	if !ok {
		// the pod doesn't need grouping schedule
		// return as it is.
		klog.V(4).Infof("pod %s/%s doesn't need grouping schedule, skip it", pod.Namespace, pod.Name)
		return returnUnmodified()
	}

	// get policy manifesting the pod
	policy, err := e.getPolicyWithNamespaceName(policyNamespaceName)
	if err != nil {
		klog.Errorf("failed to get policy %s when do filter, %v", policyNamespaceName, err)
		return nil, err
	}

	var originNodes []corev1.Node
	originNodes = append(originNodes, args.Nodes.Items...)
	// TODO:
	// register filter functions as plugins and reconstitute it with loop
	filteredNodes, err := e.filterNodesNotInClusters(pod, originNodes, policy)
	if err != nil {
		klog.Errorf("failed to filterNodesNotInClusters, %v", err)
		return returnUnmodified()
	}
	filteredNodes, err = e.filterNodesHasEnoughPods(pod, filteredNodes, policy)
	if err != nil {
		klog.Errorf("failed to filterNodesHasEnoughPods, %v", err)
		return returnUnmodified()
	}

	return e.constructFilterResult(filteredNodes, *args), nil
}

// filterNodesHasEnoughPods will filter
func (e *extender) filterNodesHasEnoughPods(pod *corev1.Pod, nodes []corev1.Node, policy *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error) {

	relativeDeploy, err := utils.GetRelativeDeployment(e.ctx, e.client, pod, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative deployment of pod %s/%s when filtering nodes for it, %v",
			pod.Namespace, pod.Name, err)
	}

	desiredPodsNumOfEachCluster := utils.DesiredPodsNumInTargetClusters(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachCluster, nodesInCluter, err := utils.CurrentPodsNumInTargetClusters(e.ctx, e.client, relativeDeploy, policy)

	if err != nil {
		return nil, fmt.Errorf("failed to get current number of pods in each target clusters for deploy %s/%s, %v",
			relativeDeploy.Namespace, relativeDeploy.Name, err)
	}

	filteredNodes := []corev1.Node{}
	for _, node := range nodes {
		cluster := nodesInCluter[node.Name]
		if currentPodsNumOfEachCluster[cluster] < desiredPodsNumOfEachCluster[cluster] {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, nil
}

// filterNodesNotInClusters will filter nodes that in the target clusters.
func (e *extender) filterNodesNotInClusters(pod *corev1.Pod, nodes []corev1.Node, policy *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error) {
	// get all target clusters
	var clusterNames []string
	for _, targetWeight := range policy.Spec.Placement.StaticWeightList {
		clusterNames = append(clusterNames, targetWeight.TargetCluster.ClusterNames...)
	}
	clusters, err := utils.GetClustersWithName(e.ctx, e.client, clusterNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters obj according to their names, err: %v", err)
	}

	// get map that map node to cluster it belongs to
	nodesInClusters, err := utils.GetNodesInClusters(e.ctx, e.client, clusters)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes in clusters when filter the nodes, %v", err)
	}

	// filter nodes that are not in target clusters
	filterdNodes := []corev1.Node{}
	for _, node := range nodes {
		if _, ok := nodesInClusters[node.Name]; ok {
			filterdNodes = append(filterdNodes, node)
		}
	}
	return filterdNodes, nil
}

func (e *extender) getPolicyWithNamespaceName(policyNamespaceName string) (*policyv1alpha1.PropagationPolicy, error) {
	policy := &policyv1alpha1.PropagationPolicy{}
	ns, name, err := utils.ParseNamespaceName(policyNamespaceName)
	if err != nil {
		klog.Errorf("failed to parse policy with namespaceName %s when do prioritize, %v", policyNamespaceName, err)
		return nil, err
	}

	if err := e.client.Get(e.ctx, client.ObjectKey{Namespace: ns, Name: name}, policy); err != nil {
		klog.Errorf("failed to get policy %s/%s, %v", ns, name, err)
		return nil, err
	}
	return policy, nil
}

func (e *extender) constructFilterResult(nodes []corev1.Node, args extenderv1.ExtenderArgs) *extenderv1.ExtenderFilterResult {
	filteredNodeNames := make([]string, len(nodes))
	for i := range nodes {
		filteredNodeNames[i] = nodes[i].Name
	}
	filterResults := &extenderv1.ExtenderFilterResult{
		Nodes: &corev1.NodeList{
			Items: nodes,
		},
		NodeNames: &filteredNodeNames,
	}

	return filterResults
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
				responseBody = nil
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
		klog.V(4).Infof("get filter request info: %v", buf.String())

		// do filter
		extenderResults, err := filterFunc(extenderArgs)
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
