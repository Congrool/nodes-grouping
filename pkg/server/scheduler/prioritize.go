package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/server/constants"
	"github.com/Congrool/nodes-grouping/pkg/utils"
)

func (e *extender) Prioritize(extenderArgs *extenderv1.ExtenderArgs) (*extenderv1.HostPriorityList, error) {
	args := extenderArgs.DeepCopy()
	pod := args.Pod
	policyNamespaceName, ok := pod.Annotations[constants.GroupingScheduleKey]
	if !ok {
		// the pod doesn't need grouping schedule
		// do not score.
		priorityList, _ := e.notScore(pod, extenderArgs, nil)
		return &priorityList, nil
	}

	policy, err := e.getPolicyWithNamespaceName(policyNamespaceName)
	if err != nil {
		klog.Errorf("failed to get policy %s when do prioritize, %v", policyNamespaceName, err)
		return nil, err
	}

	priorityList, err := e.scoreNodesAccordingDiff(pod, extenderArgs, policy)
	if err != nil {
		klog.Errorf("failed to scoreNodesAccordingDiff, %v", err)
		return nil, err
	}

	return &priorityList, nil
}

func (e *extender) notScore(pod *corev1.Pod, args *extenderv1.ExtenderArgs, _ *policyv1alpha1.PropagationPolicy) (extenderv1.HostPriorityList, error) {
	nodeNames := *args.NodeNames
	prioritList := make([]extenderv1.HostPriority, len(nodeNames))
	for _, name := range nodeNames {
		prioritList = append(prioritList, extenderv1.HostPriority{
			Host:  name,
			Score: 0,
		})
	}
	return (extenderv1.HostPriorityList)(prioritList), nil
}

type clusterItem struct {
	clusterName string
	podsNum     int
	rank        int
}

type clusterItemSlice []clusterItem

func (c clusterItemSlice) Len() int           { return len(c) }
func (c clusterItemSlice) Less(i, j int) bool { return c[i].podsNum < c[j].podsNum }
func (c clusterItemSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

// TODO:
func (e *extender) scoreNodesAccordingDiff(pod *corev1.Pod, args *extenderv1.ExtenderArgs, policy *policyv1alpha1.PropagationPolicy) (extenderv1.HostPriorityList, error) {
	relativeDeploy, err := utils.GetRelativeDeployment(e.ctx, e.client, pod, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative deployment for pod %s/%s when prioritizing nodes for it, %v",
			pod.Namespace, pod.Name, err)
	}
	desiredPodsNumOfEachCluster := utils.DesiredPodsNumInTargetClusters(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachCluster, nodesIncluster, err := utils.CurrentPodsNumInTargetClusters(e.ctx, e.client, relativeDeploy, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get current pods number in clusters for pod %s/%s with policy %s/%s, %v",
			pod.Namespace, pod.Name, policy.Namespace, policy.Name, err)
	}

	var diffList clusterItemSlice
	for cluster, desiredNum := range desiredPodsNumOfEachCluster {
		diffPodsNum := desiredNum - currentPodsNumOfEachCluster[cluster]
		diffList = append(diffList, clusterItem{clusterName: cluster, podsNum: diffPodsNum})
	}
	diffList = sort.Reverse(diffList).(clusterItemSlice)
	diffMap := make(map[string]clusterItem, diffList.Len())
	for i := range diffList {
		diffList[i].rank = i + 1
		clusterName := diffList[i].clusterName
		diffMap[clusterName] = diffList[i]
	}

	priorityList := extenderv1.HostPriorityList{}
	for _, nodename := range *args.NodeNames {
		cluster, ok := nodesIncluster[nodename]
		if !ok {
			klog.Errorf("extender prioritizer get node %s which is not in target clusters, ignore it")
			continue
		}
		ratio := (float64)(diffMap[cluster].rank) / (float64)(diffList.Len())
		score := (int64)(10 - 10*ratio)
		priorityList = append(priorityList, extenderv1.HostPriority{Host: nodename, Score: score})
	}

	return priorityList, nil
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

		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)

		// decode
		if err := json.NewDecoder(body).Decode(extenderArgs); err != nil {
			klog.Errorf("failed to decode filter extenderArgs, request: %s, %v", buf.String(), err)
			prioritizeHostList = nil
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		klog.V(4).Infof("get prioritize request info: %v", buf.String())

		// do prioritize
		prioritizeHostList, err := prioritizeFunc(extenderArgs)
		if err != nil {
			klog.Errorf("failed to run prioritize handler, err: %v", err)
			prioritizeHostList = nil
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

	})
}
