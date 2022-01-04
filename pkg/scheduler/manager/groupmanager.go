package manager

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	informerappsv1 "k8s.io/client-go/informers/apps/v1"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/klog/v2"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	informer "github.com/Congrool/nodes-grouping/pkg/generated/informers/externalversions/groupmanagement/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/scheduler/utils"
)

type GroupManager interface {
	DesiredPodsNumInTargetNodeGroups(policy *groupmanagementv1alpha1.PropagationPolicy, deploy *appsv1.Deployment) map[string]int32
	CurrentPodsNumInTargetNodeGroups(policy *groupmanagementv1alpha1.PropagationPolicy, deploy *appsv1.Deployment) (map[string]int32, error)
	GetRelativeDeployAndPolicy(pod *corev1.Pod) (*groupmanagementv1alpha1.PropagationPolicy, *appsv1.Deployment, error)
	MapNodeToNodeGroup(policy *groupmanagementv1alpha1.PropagationPolicy) map[string]string
}

var _ GroupManager = &groupManager{}

type groupManager struct {
	policyInformer    informer.PropagationPolicyInformer
	nodegroupInformer informer.NodeGroupInformer
	deployInformer    informerappsv1.DeploymentInformer
	podInformer       informercorev1.PodInformer
}

func New(policyInformer informer.PropagationPolicyInformer,
	nodegroupInformer informer.NodeGroupInformer,
	deployInformer informerappsv1.DeploymentInformer,
	podInformer informercorev1.PodInformer) GroupManager {

	return &groupManager{
		policyInformer:    policyInformer,
		nodegroupInformer: nodegroupInformer,
		deployInformer:    deployInformer,
		podInformer:       podInformer,
	}
}

func (gm *groupManager) GetRelativeDeployAndPolicy(pod *corev1.Pod) (*groupmanagementv1alpha1.PropagationPolicy, *appsv1.Deployment, error) {
	policyList, err := gm.policyInformer.Lister().List(labels.Everything())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list PropagationPolicy: %v", err)
	}

	for _, policy := range policyList {
		var deploys []*appsv1.Deployment
		for _, selector := range policy.Spec.ResourceSelectors {
			if selector.Namespace != "" && selector.Name != "" {
				deploy, err := gm.deployInformer.Lister().Deployments(selector.Namespace).Get(selector.Name)
				if err != nil {
					klog.Errorf("failed to get deploy %s/%s, %v", selector.Namespace, selector.Name)
					continue
				}
				deploys = append(deploys, deploy)
			}
		}

		for _, deploy := range deploys {
			if utils.IfPodMatchDeploy(deploy, pod) {
				return policy, deploy, nil
			}
		}
	}

	// find nothing
	return nil, nil, nil
}

func (gm *groupManager) DesiredPodsNumInTargetNodeGroups(policy *groupmanagementv1alpha1.PropagationPolicy, deploy *appsv1.Deployment) map[string]int32 {
	weights := policy.Spec.Placement.StaticWeightList
	replicaNum := *deploy.Spec.Replicas

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

func (gm *groupManager) MapNodeToNodeGroup(policy *groupmanagementv1alpha1.PropagationPolicy) map[string]string {
	// get all nodegroups relative to the policy
	targetNodeGroups := []*groupmanagementv1alpha1.NodeGroup{}
	for _, weight := range policy.Spec.Placement.StaticWeightList {
		for _, name := range weight.NodeGroupNames {
			// TODO:
			// make nodegroup non-namespaced
			nodegroup, err := gm.nodegroupInformer.Lister().NodeGroups("default").Get(name)
			if err != nil {
				klog.Errorf("failed to get nodegroup %s, %v", name, err)
				continue
			}
			targetNodeGroups = append(targetNodeGroups, nodegroup)
		}
	}

	// find all nodes in corresponding nodegroups
	nodeToNodeGroup := make(map[string]string)
	for _, nodegroup := range targetNodeGroups {
		for _, nodename := range nodegroup.Status.ContainedNodes {
			if groupname, ok := nodeToNodeGroup[nodename]; ok {
				klog.Errorf("node %s has already belonged to nodegroup %s, find it also belongs to nodegroup %s", nodename, groupname, nodegroup)
				continue
			}
			nodeToNodeGroup[nodename] = nodegroup.Name
		}
	}
	return nodeToNodeGroup
}

func (gm *groupManager) CurrentPodsNumInTargetNodeGroups(policy *groupmanagementv1alpha1.PropagationPolicy, deploy *appsv1.Deployment) (map[string]int32, error) {
	nodeToNodeGroup := gm.MapNodeToNodeGroup(policy)
	selector, err := utils.GetPodSelectorFromDeploy(deploy)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod selector from deploy: %s/%s, %v", deploy.Namespace, deploy.Name, err)
	}
	pods, err := gm.podInformer.Lister().List(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods of deploy: %s/%s, %v", deploy.Namespace, deploy.Name, err)
	}

	currentPodsInTargetNodeGroups := map[string]int32{}
	for _, pod := range pods {
		if pod.Spec.NodeName == "" {
			// ignore no scheduled pod
			continue
		}
		group, ok := nodeToNodeGroup[pod.Spec.NodeName]
		if !ok {
			// It should be solved by PropagationPolicy controller instead of the scheduler extender.
			klog.Warningf("find pod running on the node %s which is not in target nodegroups, ignore it")
			continue
		}
		currentPodsInTargetNodeGroups[group]++
	}
	return currentPodsInTargetNodeGroups, nil
}
