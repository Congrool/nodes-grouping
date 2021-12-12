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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nodegroupv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/group/v1alpha1"
	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/utils"
)

// Controller reconciles a PropagationPolicy object
type Controller struct {
	client.Client

	// TODO:
	// not used
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=policy.harmonycloud.io,resources=propagationpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (p *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	policy := &policyv1alpha1.PropagationPolicy{}
	if err := p.Client.Get(ctx, req.NamespacedName, policy); err != nil {
		if apierrors.IsNotFound(err) || policy.DeletionTimestamp != nil {
			// TODO: handle delete event
			// currently do nothing, leave the scheduled pods as they are.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, err
	}

	nodegroupList := &nodegroupv1alpha1.NodeGroupList{}
	if err := p.Client.List(ctx, nodegroupList, &client.ListOptions{}); err != nil {
		klog.Errorf("failed to list nodegroup, %v", err)
		return ctrl.Result{}, nil
	}
	nodesInNodeGroups, err := utils.GetNodesInGroups(ctx, p.Client, nodegroupList.Items)

	if err != nil {
		klog.Errorf("failed to get nodes in nodegroups, err: %v", err)
		return ctrl.Result{}, nil
	}

	// TODO:
	// Currently, only support selecting deploys with their namespace and name.
	// More approaches are needed.
	deploys, err := utils.GetManifestsDeploys(ctx, p.Client, policy)
	if err != nil {
		klog.Warningf("failed to get some deploys manifested by policy %s/%s, %v", policy.Namespace, policy.Name, err)
	}

	errs := []error{}
	for _, deploy := range deploys {
		podList, err := utils.GetPodListFromDeploy(ctx, p.Client, deploy)
		desiredPodsNumOfEachNodeGroup := utils.DesiredPodsNumInTargetNodeGroups(policy.Spec.Placement.StaticWeightList, *deploy.Spec.Replicas)
		if err != nil {
			klog.Errorf("failed to get pod list of deployment %s/%s, %v", deploy.Namespace, deploy.Name, err)
			continue
		}

		deletePods := getPodsNeedToDelete(podList.Items, desiredPodsNumOfEachNodeGroup, nodesInNodeGroups)
		for _, pod := range deletePods {
			if err := p.Client.Delete(ctx, pod); err != nil && apierrors.IsNotFound(err) {
				klog.Errorf("failed to delete pod %s/%s, %v", pod.Namespace, pod.Name, err)
				errs = append(errs, err)
			}
		}
	}

	return ctrl.Result{}, errors.NewAggregate(errs)
}

func (p *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha1.PropagationPolicy{}).
		// watch changes of NodeGroup and enqueue relavent policies
		// when nodes in node group has changed.
		Watches(&source.Kind{Type: &nodegroupv1alpha1.NodeGroup{}}, handler.EnqueueRequestsFromMapFunc(p.newNodeGroupMapFunc)).
		Complete(p)
}

// SetupWithManager sets up the controller with the Manager.
func (p *Controller) newNodeGroupMapFunc(obj client.Object) []ctrl.Request {
	groupobj := obj.(*nodegroupv1alpha1.NodeGroup)
	policyList := &policyv1alpha1.PropagationPolicyList{}
	if err := p.Client.List(context.TODO(), policyList); err != nil {
		klog.Errorf("failed to list propagation policy, %v", err)
		return nil
	}

	results := []ctrl.Request{}

	forEachPolicyDo := func(fn func(*policyv1alpha1.PropagationPolicy)) {
		for i := range policyList.Items {
			fn(&policyList.Items[i])
		}
	}
	ifNodeGroupInPolicy := func(policy *policyv1alpha1.PropagationPolicy) bool {
		for _, weight := range policy.Spec.Placement.StaticWeightList {
			for _, group := range weight.NodeGroupNames {
				if group == groupobj.Name {
					return true
				}
			}
		}
		return false
	}

	forEachPolicyDo(func(policy *policyv1alpha1.PropagationPolicy) {
		if ifNodeGroupInPolicy(policy) {
			results = append(results, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: policy.Namespace,
					Name:      policy.Name,
				}})
		}
	})

	return results
}

func getPodsNeedToDelete(pods []corev1.Pod, desiredPods map[string]int, nodesInNodeGroups map[string]string) []*corev1.Pod {
	deletePod := []*corev1.Pod{}
	count := make(map[string]int)

	for _, pod := range pods {
		if pod.Spec.NodeName == "" {
			continue
		}
		if groupname, ok := nodesInNodeGroups[pod.Spec.NodeName]; ok {
			if count[groupname] > desiredPods[groupname] {
				// More than desired number of pods can run in this nodegroup
				deletePod = append(deletePod, &pod)
			}
			count[groupname]++
		}
	}
	return deletePod
}
