package scheduler

import (
	"context"
	"fmt"
	"sort"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/utils"
)

type nodeGroupItem struct {
	nodeGroupName string
	podsNum       int32
	rank          int
}

type nodeGroupItemSlice []nodeGroupItem

func (c nodeGroupItemSlice) Len() int           { return len(c) }
func (c nodeGroupItemSlice) Less(i, j int) bool { return c[i].podsNum < c[j].podsNum }
func (c nodeGroupItemSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type NodeGroupScheduling struct {
	client client.Client
}

const (
	Name         = "NodeGroupScheduling"
	NodeGroupKey = "nodegroup"
)

type nodeGroupStateData struct {
	policyNamespace string
	policyName      string
	policy          *policyv1alpha1.PropagationPolicy
	deploy          *appv1.Deployment
}

func (nsd *nodeGroupStateData) Clone() framework.StateData {
	return &nodeGroupStateData{
		policyNamespace: nsd.policyNamespace,
		policyName:      nsd.policyName,
		policy:          nsd.policy,
	}
}

var _ framework.PreFilterPlugin = &NodeGroupScheduling{}
var _ framework.FilterPlugin = &NodeGroupScheduling{}
var _ framework.ScorePlugin = &NodeGroupScheduling{}

func (ngs *NodeGroupScheduling) Name() string {
	return Name
}

func (ngs *NodeGroupScheduling) PreFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod) *framework.Status {
	deploy, policy, err := utils.GetRelativeDeployAndPolicy(ctx, ngs.client, pod)
	if err != nil {
		return framework.NewStatus(
			framework.Success,
			fmt.Sprintf("failed to get relative policy for pod %s/%s, %v, fall back to normal schedule",
				pod.Namespace, pod.Name, err),
		)
	}

	if policy == nil {
		// no relative policy
		// schedule this pod in normal way
		return framework.NewStatus(framework.Success, fmt.Sprintf("no relative policy find for pod %s/%s, fall back to normal schedule",
			pod.Namespace, pod.Name))
	}

	state.Write(NodeGroupKey, &nodeGroupStateData{
		policyNamespace: policy.Namespace,
		policyName:      policy.Name,
		policy:          policy,
		deploy:          deploy,
	})

	return framework.NewStatus(framework.Success, fmt.Sprintf("find policy %s/%s for pod %s/%s", policy.Namespace, policy.Name, pod.Namespace, pod.Name))
}

func (ngs *NodeGroupScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ngs *NodeGroupScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	var policyData *nodeGroupStateData
	if data, err := state.Read(NodeGroupKey); err != nil {
		// fall back to normal pod scheduling
		return framework.NewStatus(framework.Success, "")
	} else if d, ok := data.(*nodeGroupStateData); !ok {
		return framework.NewStatus(framework.Success, fmt.Sprintf("convert StateDate to nodeGroupStateData failed, fall back to normal schedule"))
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := utils.DesiredPodsNumInTargetNodeGroups(policy.Spec.Placement.StaticWeightList, *deploy.Spec.Replicas)
	currentPodsNumOfEachNodeGroup, nodesInNodeGroup, err := utils.CurrentPodsNumInTargetNodeGroups(ctx, ngs.client, deploy, policy)
	if err != nil {
		return framework.NewStatus(framework.Success, fmt.Sprintf("CurrentPodsNumInTargetNodeGroups failed: %v, fall back to normal schedule", err))
	}

	nodegroup, find := nodesInNodeGroup[nodeInfo.Node().Name]
	if !find {
		// This node is not in any nodegroup
		return framework.NewStatus(framework.Unschedulable, "")
	}

	desciredPodsNum, currentPodsNum := desiredPodsNumOfEachNodeGroup[nodegroup], currentPodsNumOfEachNodeGroup[nodegroup]
	if currentPodsNum >= desciredPodsNum {
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("nodegroup %s already has enough pod num %d", nodegroup, currentPodsNum))
	}
	return framework.NewStatus(framework.Success, "")
}

func (ngs *NodeGroupScheduling) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodename string) (int64, *framework.Status) {
	var policyData *nodeGroupStateData
	if data, err := state.Read(NodeGroupKey); err != nil {
		// fall back to normal pod scheduling
		return 0, framework.NewStatus(framework.Success, "")
	} else if d, ok := data.(*nodeGroupStateData); !ok {
		return 0, framework.NewStatus(framework.Success, fmt.Sprintf("convert StateDate to nodeGroupStateData failed, fall back to normal schedule"))
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := utils.DesiredPodsNumInTargetNodeGroups(policy.Spec.Placement.StaticWeightList, *deploy.Spec.Replicas)
	currentPodsNumOfEachNodeGroup, nodesInNodeGroup, err := utils.CurrentPodsNumInTargetNodeGroups(ctx, ngs.client, deploy, policy)
	if err != nil {
		return 0, framework.NewStatus(framework.Success, fmt.Sprintf("CurrentPodsNumInTargetNodeGroups failed: %v, fall back to normal schedule", err))
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

	if nodegroup, ok := nodesInNodeGroup[nodename]; ok {
		ratio := (float64)(diffMap[nodegroup].rank) / (float64)(diffList.Len())
		score := (int64)(10 - 10*ratio)
		return score, framework.NewStatus(framework.Success, "")
	}
	return 0, framework.NewStatus(framework.Error,
		fmt.Sprintf("failed find node %s in any nodegroup, this should not happen in score plugin after filer running expectedly", nodename))
}

func (ngs *NodeGroupScheduling) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
