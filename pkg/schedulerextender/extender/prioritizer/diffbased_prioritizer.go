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

type nodeGroupItem struct {
	nodeGroupName string
	podsNum       int32
	rank          int
}

type nodeGroupItemSlice []nodeGroupItem

func (c nodeGroupItemSlice) Len() int           { return len(c) }
func (c nodeGroupItemSlice) Less(i, j int) bool { return c[i].podsNum < c[j].podsNum }
func (c nodeGroupItemSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

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
	desiredPodsNumOfEachNodeGroup := utils.DesiredPodsNumInTargetNodeGroups(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachNodeGroup, nodesInNodeGroup, err := utils.CurrentPodsNumInTargetNodeGroups(ctx, client, relativeDeploy, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get current pods number in nodegroup for pod %s/%s with policy %s/%s, %v",
			pod.Namespace, pod.Name, policy.Namespace, policy.Name, err)
	}

	var diffList nodeGroupItemSlice
	for nodegroup, desiredNum := range desiredPodsNumOfEachNodeGroup {
		diffPodsNum := desiredNum - currentPodsNumOfEachNodeGroup[nodegroup]
		diffList = append(diffList, nodeGroupItem{nodeGroupName: nodegroup, podsNum: diffPodsNum})
	}
	diffList = sort.Reverse(diffList).(nodeGroupItemSlice)
	diffMap := make(map[string]nodeGroupItem, diffList.Len())
	for i := range diffList {
		diffList[i].rank = i + 1
		nodeGroupName := diffList[i].nodeGroupName
		diffMap[nodeGroupName] = diffList[i]
	}

	priorityList := extenderv1.HostPriorityList{}
	for _, nodename := range *args.NodeNames {
		nodeGroup, ok := nodesInNodeGroup[nodename]
		if !ok {
			klog.Errorf("extender prioritizer get node %s which is not in target nodegroup, ignore it")
			continue
		}
		ratio := (float64)(diffMap[nodeGroup].rank) / (float64)(diffList.Len())
		score := (int64)(10 - 10*ratio)
		priorityList = append(priorityList, extenderv1.HostPriority{Host: nodename, Score: score})
	}

	return priorityList, nil
}
