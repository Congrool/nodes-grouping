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

var _ FilterPlugin = &notInNodeGroupsFilter{}

// filterNodesNotInNodeGroups will filter nodes that not in the target nodegroups.
type notInNodeGroupsFilter struct{}

func (f *notInNodeGroupsFilter) Name() string {
	return constants.NotInNodeGroupsFilterPluginName
}

func (f *notInNodeGroupsFilter) FilterNodes(
	ctx context.Context,
	client client.Client,
	pod *corev1.Pod,
	nodes []corev1.Node,
	policy *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error) {
	// get all target nodegroups
	var nodeGroupNames []string
	for _, targetWeight := range policy.Spec.Placement.StaticWeightList {
		nodeGroupNames = append(nodeGroupNames, targetWeight.NodeGroupNames...)
	}
	nodegroups, err := utils.GetNodeGroupsWithName(ctx, client, nodeGroupNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodegroup obj according to their names, err: %v", err)
	}

	// get map that map node to nodegroup it belongs to
	nodesInGroups, err := utils.GetNodesInGroups(ctx, client, nodegroups)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes in nodegroup when filter the nodes, %v", err)
	}

	// filter nodes that are not in target nodegroups
	filterdNodes := []corev1.Node{}
	for _, node := range nodes {
		if _, ok := nodesInGroups[node.Name]; ok {
			filterdNodes = append(filterdNodes, node)
		}
	}
	return filterdNodes, nil
}
