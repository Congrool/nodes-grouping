package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	clusterv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func WithCheck(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "Empty Request Body", 400)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func GetNodesInClusters(ctx context.Context, client runtimeClient.Client, clusters []clusterv1alpha1.Cluster) (map[string]string, error) {
	nodesInClusters := make(map[string]string)
	for _, cluster := range clusters {
		labelSelector := metav1.SetAsLabelSelector(cluster.Spec.MatchLabels)
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			klog.Errorf("failed to get list selector according to matchLabels of cluster: %s, err %v", cluster.Name, err)
			return nil, err
		}
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList, &runtimeClient.ListOptions{LabelSelector: selector}); err != nil {
			klog.Errorf("failed to list node for cluster %s, %v", cluster.ClusterName, err)
			return nil, err
		}
		for i := range nodeList.Items {
			nodesInClusters[nodeList.Items[i].Name] = cluster.ClusterName
		}
	}

	return nodesInClusters, nil
}

func GetClustersWithName(ctx context.Context, client runtimeClient.Client, clusterName []string) ([]clusterv1alpha1.Cluster, error) {
	clusters := []clusterv1alpha1.Cluster{}
	for _, name := range clusterName {
		cluster := &clusterv1alpha1.Cluster{}
		if err := client.Get(ctx, runtimeClient.ObjectKey{Name: name}, cluster); err != nil {
			klog.Errorf("failed to get cluster obj %s, %v", name, err)
			return nil, err
		}
		clusters = append(clusters, *cluster)
	}
	return clusters, nil
}

func ParseNamespaceName(namespaceName string) (string, string, error) {
	keys := strings.Split(namespaceName, "/")
	if len(keys) == 1 {
		return "default", keys[0], nil
	}
	if len(keys) == 2 {
		return keys[0], keys[1], nil
	}
	return "", "", errors.New(fmt.Sprintf("failed to parse NamespaceName of %s", namespaceName))
}
