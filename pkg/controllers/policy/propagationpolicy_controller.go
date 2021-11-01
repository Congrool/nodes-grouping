/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policy

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/cluster/v1alpha1"
	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/utils"
)

// PropagationPolicyReconciler reconciles a PropagationPolicy object
type PropagationPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PropagationPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *PropagationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	policy := &policyv1alpha1.PropagationPolicy{}
	if err := r.Client.Get(ctx, req.NamespacedName, policy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	clusterList := &clusterv1alpha1.ClusterList{}
	if err := r.Client.List(ctx, clusterList, &client.ListOptions{}); err != nil {
		klog.Errorf("failed to list cluster, %v", err)
		return ctrl.Result{}, nil
	}
	nodesInClusters, err := utils.GetNodesInClusters(ctx, r.Client, clusterList.Items)

	if err != nil {
		klog.Errorf("failed to get nodes in clusters, err: %v", err)
		return ctrl.Result{}, nil
	}

	deploys := r.fetchManifestsResource(ctx, policy)

	errs := []error{}
	for _, deploy := range deploys {
		podList, err := r.getPodListFromDeploy(ctx, deploy)
		desiredClusterAndPods := desiredPodsNumInEachCluster(policy.Spec.Placement.StaticWeightList, *deploy.Spec.Replicas)
		if err != nil {
			klog.Errorf("failed to get pod list of deployment %s/%s, %v", deploy.Namespace, deploy.Name, err)
			continue
		}

		deletePods := r.getPodsNeedToDelete(podList.Items, desiredClusterAndPods, nodesInClusters)
		for _, pod := range deletePods {
			if err := r.Client.Delete(ctx, pod); err != nil {
				klog.Errorf("failed to delete pod %s/%s, %v", pod.Namespace, pod.Name, err)
				errs = append(errs, err)
			}
		}
	}

	return ctrl.Result{}, errors.NewAggregate(errs)
}

func (r *PropagationPolicyReconciler) getPodListFromDeploy(ctx context.Context, deploy *appsv1.Deployment) (*corev1.PodList, error) {
	labelselector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList := &corev1.PodList{}
	if err := r.Client.List(ctx, podList, &client.ListOptions{LabelSelector: labelselector}); err != nil {
		return nil, err
	}
	return podList, nil
}

func (r *PropagationPolicyReconciler) getPodsNeedToDelete(pods []corev1.Pod, desiredPods map[string]int, nodesInClusters map[string]string) []*corev1.Pod {
	deletePod := []*corev1.Pod{}
	count := make(map[string]int)

	for _, pod := range pods {
		if pod.Spec.NodeName == "" {
			continue
		}
		if clusterName, ok := nodesInClusters[pod.Spec.NodeName]; ok {
			if count[clusterName] >= desiredPods[clusterName] {
				// More than desired number of pods can run in this cluster
				deletePod = append(deletePod, &pod)
			}
			count[clusterName]++
		}
	}
	return deletePod
}

// SetupWithManager sets up the controller with the Manager.
func (r *PropagationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha1.PropagationPolicy{}).
		Watches(&source.Kind{Type: &clusterv1alpha1.Cluster{}}, handler.EnqueueRequestsFromMapFunc(r.newClusterMapFunc)).
		Complete(r)
}

func (r *PropagationPolicyReconciler) fetchManifestsResource(ctx context.Context, policy *policyv1alpha1.PropagationPolicy) []*appsv1.Deployment {
	deploys := []*appsv1.Deployment{}
	for _, selector := range policy.Spec.ResourceSelectors {
		deploy := &appsv1.Deployment{}
		key := types.NamespacedName{Namespace: selector.Namespace, Name: selector.Name}
		err := r.Client.Get(ctx, key, deploy)
		if err != nil {
			klog.Errorf("failed to get deployment namespace: %s name: %s, %v", selector.Namespace, selector.Name)
			continue
		}
		deploys = append(deploys, deploy)
	}
	return deploys
}

func (p *PropagationPolicyReconciler) newClusterMapFunc(obj client.Object) []ctrl.Request {
	clusterobj := obj.(*clusterv1alpha1.Cluster)
	policyList := &policyv1alpha1.PropagationPolicyList{}
	if err := p.Client.List(context.TODO(), policyList); err != nil {
		klog.Errorf("failed to list propagation policy, %v", err)
		return nil
	}

	results := []ctrl.Request{}

	forEachPolicy := func(fn func(*policyv1alpha1.PropagationPolicy)) {
		for i := range policyList.Items {
			fn(&policyList.Items[i])
		}
	}
	ifContainsCluster := func(policy *policyv1alpha1.PropagationPolicy) bool {
		for _, weight := range policy.Spec.Placement.StaticWeightList {
			for _, cluster := range weight.TargetCluster.ClusterNames {
				if cluster == clusterobj.Name {
					return true
				}
			}
		}
		return false
	}

	forEachPolicy(func(policy *policyv1alpha1.PropagationPolicy) {
		if ifContainsCluster(policy) {
			results = append(results, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: policy.Namespace,
					Name:      policy.Name,
				}})
		}
	})

	return results
}

func desiredPodsNumInEachCluster(weights []policyv1alpha1.StaticClusterWeight, replicaNum int32) map[string]int {
	var sum int64
	results := make(map[string]int)
	for _, weight := range weights {
		for range weight.TargetCluster.ClusterNames {
			sum += weight.Weight
		}
	}

	for _, weight := range weights {
		ratio := float64(weight.Weight) / float64(sum)
		desiredNum := int(ratio*float64(replicaNum) + 0.5)
		for _, cluster := range weight.TargetCluster.ClusterNames {
			results[cluster] = desiredNum
		}
	}

	return results
}
