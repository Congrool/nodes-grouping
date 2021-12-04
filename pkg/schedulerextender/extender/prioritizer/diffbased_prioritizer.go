package prioritizer

import (
	"context"
	"fmt"
	"sort"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/constants"
	"github.com/Congrool/nodes-grouping/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ PrioritizerPlugin = &diffBasedPrioritizePlugin{}

type clusterItem struct {
	clusterName string
	podsNum     int
	rank        int
}

type clusterItemSlice []clusterItem

func (c clusterItemSlice) Len() int           { return len(c) }
func (c clusterItemSlice) Less(i, j int) bool { return c[i].podsNum < c[j].podsNum }
func (c clusterItemSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type diffBasedPrioritizePlugin struct{}

func (p *diffBasedPrioritizePlugin) Name() string {
	return constants.DiffBasedPrioritizePluginName
}

// TODO:
// figure out how to score nodes when the number of the candicates is too small.
func (p *diffBasedPrioritizePlugin) PrioritizeNodes(ctx context.Context, client client.Client, pod *corev1.Pod, args *extenderv1.ExtenderArgs, policy *policyv1alpha1.PropagationPolicy) (extenderv1.HostPriorityList, error) {
	relativeDeploy, err := utils.GetRelativeDeployment(ctx, client, pod, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative deployment for pod %s/%s when prioritizing nodes for it, %v",
			pod.Namespace, pod.Name, err)
	}
	desiredPodsNumOfEachCluster := utils.DesiredPodsNumInTargetClusters(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachCluster, nodesIncluster, err := utils.CurrentPodsNumInTargetClusters(ctx, client, relativeDeploy, policy)
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
