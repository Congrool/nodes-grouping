package filter

import (
	"context"
	"fmt"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/constants"
	"github.com/Congrool/nodes-grouping/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ FilterPlugin = &enoughPodsFilter{}

type enoughPodsFilter struct{}

func (f *enoughPodsFilter) Name() string {
	return constants.EnoughPodsFilterPluginName
}

// filterNodesHasEnoughPods will filter nodes in nodegroups
// which have already had enough pods as desired in the policy.
func (f *enoughPodsFilter) FilterNodes(
	ctx context.Context,
	client client.Client,
	pod *corev1.Pod,
	nodes []corev1.Node,
	policy *groupmanagementv1alpha1.PropagationPolicy) ([]corev1.Node, error) {
	relativeDeploy, err := utils.GetRelativeDeployment(ctx, client, pod, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative deployment of pod %s/%s when filtering nodes for it, %v",
			pod.Namespace, pod.Name, err)
	}

	desiredPodsNumOfEachNodeGroup := utils.DesiredPodsNumInTargetNodeGroups(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachNodeGroup, nodesInNodeGroup, err := utils.CurrentPodsNumInTargetNodeGroups(ctx, client, relativeDeploy, policy)

	if err != nil {
		return nil, fmt.Errorf("failed to get current number of pods in each target nodegroups for deploy %s/%s, %v",
			relativeDeploy.Namespace, relativeDeploy.Name, err)
	}

	filteredNodes := []corev1.Node{}
	for _, node := range nodes {
		nodegroup := nodesInNodeGroup[node.Name]
		if currentPodsNumOfEachNodeGroup[nodegroup] < desiredPodsNumOfEachNodeGroup[nodegroup] {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, nil
}
