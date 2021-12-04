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

var _ FilterPlugin = &notInClustersFilter{}

type notInClustersFilter struct{}

func (f *notInClustersFilter) Name() string {
	return constants.NotInClustersFilterPluginName
}

// filterNodesNotInClusters will filter nodes that not in the target clusters.
func (f *notInClustersFilter) FilterNodes(
	ctx context.Context,
	client client.Client,
	pod *corev1.Pod,
	nodes []corev1.Node,
	policy *policyv1alpha1.PropagationPolicy) ([]corev1.Node, error) {
	// get all target clusters
	var clusterNames []string
	for _, targetWeight := range policy.Spec.Placement.StaticWeightList {
		clusterNames = append(clusterNames, targetWeight.ClusterNames...)
	}
	clusters, err := utils.GetClustersWithName(ctx, client, clusterNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters obj according to their names, err: %v", err)
	}

	// get map that map node to cluster it belongs to
	nodesInClusters, err := utils.GetNodesInClusters(ctx, client, clusters)
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
