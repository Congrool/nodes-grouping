package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apierr "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func WithCheck(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "Empty Request Body", http.StatusBadRequest)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func GetNodesInGroups(ctx context.Context, client runtimeClient.Client, groups []groupmanagementv1alpha1.NodeGroup) (map[string]string, error) {
	nodesInGroups := make(map[string]string)
	for _, group := range groups {
		labelSelector := metav1.SetAsLabelSelector(group.Spec.MatchLabels)
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			klog.Errorf("failed to get list selector according to matchLabels of nodegroup: %s, err %v", group.Name, err)
			return nil, err
		}
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList, &runtimeClient.ListOptions{LabelSelector: selector}); err != nil {
			klog.Errorf("failed to list node for nodegroup %s, %v", group.Name, err)
			return nil, err
		}
		for i := range nodeList.Items {
			nodesInGroups[nodeList.Items[i].Name] = group.Name
		}
	}

	return nodesInGroups, nil
}

func GetNodeGroupsWithName(ctx context.Context, client runtimeClient.Client, nodeGroupName []string) ([]groupmanagementv1alpha1.NodeGroup, error) {
	nodegroup := []groupmanagementv1alpha1.NodeGroup{}
	for _, name := range nodeGroupName {
		group := &groupmanagementv1alpha1.NodeGroup{}
		// TODO:
		// do not use "default" as nodegroup namespace"
		if err := client.Get(ctx, runtimeClient.ObjectKey{Name: name, Namespace: "default"}, group); err != nil {
			klog.Errorf("failed to get group obj %s, %v", name, err)
			return nil, err
		}
		nodegroup = append(nodegroup, *group)
	}
	return nodegroup, nil
}

func GetManifestsDeploys(ctx context.Context, client runtimeClient.Client, policy *groupmanagementv1alpha1.PropagationPolicy) ([]*appsv1.Deployment, error) {
	deploys := []*appsv1.Deployment{}
	errs := []error{}
	for _, selector := range policy.Spec.ResourceSelectors {
		deploy := &appsv1.Deployment{}
		key := types.NamespacedName{Namespace: selector.Namespace, Name: selector.Name}
		err := client.Get(ctx, key, deploy)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get deployment namespace: %s name: %s, %v", selector.Namespace, selector.Name, err))
			continue
		}
		deploys = append(deploys, deploy)
	}
	return deploys, apierr.NewAggregate(errs)
}

func ParseNamespaceName(namespaceName string) (string, string, error) {
	keys := strings.Split(namespaceName, "/")
	if len(keys) == 1 {
		return "default", keys[0], nil
	}
	if len(keys) == 2 {
		return keys[0], keys[1], nil
	}
	return "", "", fmt.Errorf("failed to parse NamespaceName of %s", namespaceName)
}

func DesiredPodsNumInTargetNodeGroups(weights []groupmanagementv1alpha1.StaticNodeGroupWeight, replicaNum int32) map[string]int32 {
	var sum int64
	results := make(map[string]int32)

	if len(weights) == 0 {
		return results
	}

	for _, weight := range weights {
		sum += weight.Weight
	}

	var allocatedPodNum int32
	for _, weight := range weights {
		var ratio float64
		if sum != 0 {
			ratio = float64(weight.Weight) / float64(sum)
		} else {
			ratio = 0
		}

		desiredNum := int32(ratio*float64(replicaNum) + 0.5)
		results[weight.NodeGroupNames[0]] = desiredNum
		if len(weight.NodeGroupNames) > 1 {
			// TODO:
			// support multi-nodegroup one entry
			klog.Error("multi nodegroup in one weight entry is not supported, only the first one will be picked, other nodegroup will get 0 weight.")
			for i := 2; i < len(weight.NodeGroupNames); i++ {
				results[weight.NodeGroupNames[i]] = 0
			}
		}

		allocatedPodNum += desiredNum
	}

	// TODO:
	// consider how to allocate left pods when (replicaNum % sum != 0)
	// currently add all of them to one of the nodegroups.
	leftPodNum := replicaNum - allocatedPodNum
	if leftPodNum != 0 {
		for nodegroup := range results {
			results[nodegroup] += leftPodNum
			break
		}
	}

	return results
}

func GetPodListFromDeploy(ctx context.Context, client runtimeClient.Client, deploy *appsv1.Deployment) (*corev1.PodList, error) {
	labelselector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList := &corev1.PodList{}
	if err := client.List(ctx, podList, &runtimeClient.ListOptions{LabelSelector: labelselector}); err != nil {
		return nil, err
	}
	return podList, nil
}
