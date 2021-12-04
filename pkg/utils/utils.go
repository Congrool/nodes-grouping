package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	groupv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/group/v1alpha1"
	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	apierr "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/klog/v2"
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

func GetNodesInClusters(ctx context.Context, client runtimeClient.Client, groups []groupv1alpha1.NodeGroup) (map[string]string, error) {
	nodesInClusters := make(map[string]string)
	for _, group := range groups {
		labelSelector := metav1.SetAsLabelSelector(group.Spec.MatchLabels)
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			klog.Errorf("failed to get list selector according to matchLabels of cluster: %s, err %v", group.Name, err)
			return nil, err
		}
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList, &runtimeClient.ListOptions{LabelSelector: selector}); err != nil {
			klog.Errorf("failed to list node for nodegroup %s, %v", group.Name, err)
			return nil, err
		}
		for i := range nodeList.Items {
			nodesInClusters[nodeList.Items[i].Name] = group.Name
		}
	}

	return nodesInClusters, nil
}

func GetClustersWithName(ctx context.Context, client runtimeClient.Client, clusterName []string) ([]groupv1alpha1.NodeGroup, error) {
	clusters := []groupv1alpha1.NodeGroup{}
	for _, name := range clusterName {
		cluster := &groupv1alpha1.NodeGroup{}
		if err := client.Get(ctx, runtimeClient.ObjectKey{Name: name}, cluster); err != nil {
			klog.Errorf("failed to get cluster obj %s, %v", name, err)
			return nil, err
		}
		clusters = append(clusters, *cluster)
	}
	return clusters, nil
}

func GetManifestsDeploys(ctx context.Context, client runtimeClient.Client, policy *policyv1alpha1.PropagationPolicy) ([]*appsv1.Deployment, error) {
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

func DesiredPodsNumInTargetClusters(weights []policyv1alpha1.StaticClusterWeight, replicaNum int32) map[string]int {
	var sum int64
	results := make(map[string]int)
	for _, weight := range weights {
		for range weight.ClusterNames {
			sum += weight.Weight
		}
	}

	for _, weight := range weights {
		ratio := float64(weight.Weight) / float64(sum)
		desiredNum := int(ratio*float64(replicaNum) + 0.5)
		for _, cluster := range weight.ClusterNames {
			results[cluster] = desiredNum
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

func CurrentPodsNumInTargetClusters(ctx context.Context, client runtimeClient.Client, deploy *appsv1.Deployment, policy *policyv1alpha1.PropagationPolicy) (map[string]int, map[string]string, error) {
	targetClusterNames := []string{}
	for _, weight := range policy.Spec.Placement.StaticWeightList {
		targetClusterNames = append(targetClusterNames, weight.ClusterNames...)
	}

	clusters, err := GetClustersWithName(ctx, client, targetClusterNames)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get clusters according to their names for deploy %s/%s, policy %s/%s , %v",
			deploy.Namespace, deploy.Name, policy.Namespace, policy.Name, err)
	}

	nodesInClusters, err := GetNodesInClusters(ctx, client, clusters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get nodes in clusters for deploy %s/%s, policy %s/%s, %v",
			deploy.Namespace, deploy.Name, policy.Namespace, policy.Name, err)
	}

	podList, err := GetPodListFromDeploy(ctx, client, deploy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get podlist for deploy %s/%s", deploy.Namespace, deploy.Name)
	}

	currentPodsInTargetClusters := map[string]int{}
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == "" {
			// ignore no scheduled pod
			continue
		}
		cluster, ok := nodesInClusters[pod.Spec.NodeName]
		if !ok {
			// It should be solved by PropagationPolicy controller instead of the scheduler extender.
			klog.Warningf("find pod running on the node %s which is not in target clusters, ignore it")
			continue
		}
		currentPodsInTargetClusters[cluster]++
	}
	return currentPodsInTargetClusters, nodesInClusters, nil
}

func GetRelativeDeployment(ctx context.Context, client runtimeClient.Client, pod *corev1.Pod, policy *policyv1alpha1.PropagationPolicy) (*appsv1.Deployment, error) {
	deploys, err := GetManifestsDeploys(ctx, client, policy)
	if err != nil {
		klog.Warningf("failed to get all deploys manifested by policy %s/%s, continue with fetched deploys, %v", policy.Namespace, policy.Name, err)
	}

	// get deployment relative to the pod
	var relativeDeploy *v1.Deployment
	for _, deploy := range deploys {
		selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to convert LabelSelector to Selector, %v", err)
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			relativeDeploy = deploy
			break
		}
	}

	return relativeDeploy, err
}

func GetRelativeDeployAndPolicy(ctx context.Context, client runtimeClient.Client, pod *corev1.Pod) (*appsv1.Deployment, *policyv1alpha1.PropagationPolicy, error) {
	// TODO:
	// Do not fetch directly from APIServer
	policyList := &policyv1alpha1.PropagationPolicyList{}
	if err := client.List(ctx, policyList, &runtimeClient.ListOptions{Namespace: pod.Namespace}); err != nil {
		return nil, nil, fmt.Errorf("failed to list policy, %v", err)
	}

	matchLabelsOnPod := func(deploySelector metav1.LabelSelector, podLabels map[string]string) bool {
		selector, err := metav1.LabelSelectorAsSelector(&deploySelector)
		if err != nil {
			klog.Errorf("failed to get selector from labelselector, err: %s")
			return false
		}
		if selector.Matches(labels.Set(podLabels)) {
			return true
		}
		return false
	}

	for _, policy := range policyList.Items {
		var deploys []*appsv1.Deployment
		for _, selector := range policy.Spec.ResourceSelectors {
			if selector.Namespace != "" && selector.Name != "" {
				deploy := &appsv1.Deployment{}
				if err := client.Get(ctx, runtimeClient.ObjectKey{Namespace: selector.Namespace, Name: selector.Name}, deploy); err != nil {
					return nil, nil, fmt.Errorf("failed to get deploy with namespaceName %s/%s, %v", selector.Namespace, selector.Name, err)
				}
				deploys = append(deploys, deploy)
			} else {
				return nil, nil, fmt.Errorf("empty namespace name of resource selector in policy %s/%s", policy.Namespace, policy.Name)
			}
		}

		for _, deploy := range deploys {
			if matchLabelsOnPod(*deploy.Spec.Selector, pod.Labels) {
				return deploy, &policy, nil
			}
		}
	}

	// find nothing
	return nil, nil, nil
}
