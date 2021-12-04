package filter

import (
	"context"
	"fmt"

	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
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

// filterNodesHasEnoughPods will filter nodes in clusters
// which have already had enough pods as desired in the policy.
func (f *enoughPodsFilter) FilterNodes(
	ctx context.Context,
	client client.Client,
	pod *corev1.Pod,
	nodes []corev1.Node,
	policy *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error) {
	relativeDeploy, err := utils.GetRelativeDeployment(ctx, client, pod, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative deployment of pod %s/%s when filtering nodes for it, %v",
			pod.Namespace, pod.Name, err)
	}

	desiredPodsNumOfEachCluster := utils.DesiredPodsNumInTargetClusters(policy.Spec.Placement.StaticWeightList, *relativeDeploy.Spec.Replicas)
	currentPodsNumOfEachCluster, nodesInCluter, err := utils.CurrentPodsNumInTargetClusters(ctx, client, relativeDeploy, policy)

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
