package scheduler

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	informerv1 "k8s.io/client-go/informers/apps/v1"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	config "github.com/Congrool/nodes-grouping/pkg/apis/config/v1alpha1"
	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	groupmanagementclientset "github.com/Congrool/nodes-grouping/pkg/generated/clientset/versioned"
	groupinformerfactory "github.com/Congrool/nodes-grouping/pkg/generated/informers/externalversions"
	groupmanagementinformer "github.com/Congrool/nodes-grouping/pkg/generated/informers/externalversions/groupmanagement/v1alpha1"
	util "github.com/Congrool/nodes-grouping/pkg/scheduler/utils"
)

type nodeGroupItem struct {
	nodeGroupName string
	podsNum       int32
	rank          int
}

type nodeGroupItemSlice []nodeGroupItem

func (c nodeGroupItemSlice) Len() int           { return len(c) }
func (c nodeGroupItemSlice) Less(i, j int) bool { return c[i].podsNum < c[j].podsNum }
func (c nodeGroupItemSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type NodeGroupScheduling struct {

	// policyManager     manager.PolicyManager
	// deploymentManager manager.DeploymentManager
	// nodeGroupManager  manager.NodeGroupManager
	policyInformer   groupmanagementinformer.PropagationPolicyInformer
	deployInformer   informerv1.DeploymentInformer
	podInformer      informercorev1.PodInformer
	nodgroupInformer groupmanagementinformer.NodeGroupInformer
}

const (
	Name         = "NodeGroupScheduling"
	NodeGroupKey = "nodegroup"
)

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.NodeGroupSchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("NodeGroupScheduling wants args to be of type NodeGroupSchedulingArgs, got %T", obj)
	}

	conf, err := clientcmd.BuildConfigFromFlags("", args.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init rest.Config: %v", err)
	}

	clientset := groupmanagementclientset.NewForConfigOrDie(conf)
	informerFactory := groupinformerfactory.NewSharedInformerFactory(clientset, 0)
	policyInformer := informerFactory.Groupmanagement().V1alpha1().PropagationPolicies()
	nodegroupInformer := informerFactory.Groupmanagement().V1alpha1().NodeGroups()
	podInformer := handle.SharedInformerFactory().Core().V1().Pods()
	deployInformer := handle.SharedInformerFactory().Apps().V1().Deployments()

	return &NodeGroupScheduling{
		policyInformer:   policyInformer,
		nodgroupInformer: nodegroupInformer,
		podInformer:      podInformer,
		deployInformer:   deployInformer,
	}, nil
}

type nodeGroupStateData struct {
	policyNamespace string
	policyName      string
	policy          *groupmanagementv1alpha1.PropagationPolicy
	deploy          *appsv1.Deployment
}

func (nsd *nodeGroupStateData) Clone() framework.StateData {
	return &nodeGroupStateData{
		policyNamespace: nsd.policyNamespace,
		policyName:      nsd.policyName,
		policy:          nsd.policy,
	}
}

var _ framework.PreFilterPlugin = &NodeGroupScheduling{}
var _ framework.FilterPlugin = &NodeGroupScheduling{}
var _ framework.ScorePlugin = &NodeGroupScheduling{}

func (ngs *NodeGroupScheduling) Name() string {
	return Name
}

func (ngs *NodeGroupScheduling) getRelativeDeployAndPolicy(pod *corev1.Pod) (*appsv1.Deployment, *groupmanagementv1alpha1.PropagationPolicy, error) {
	policyList, err := ngs.policyInformer.Lister().List(labels.Everything())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list PropagationPolicy: %v", err)
	}

	for _, policy := range policyList {
		var deploys []*appsv1.Deployment
		for _, selector := range policy.Spec.ResourceSelectors {
			if selector.Namespace != "" && selector.Name != "" {
				deploy, err := ngs.deployInformer.Lister().Deployments(selector.Namespace).Get(selector.Name)
				if err != nil {
					klog.Errorf("failed to get deploy %s/%s, %v", selector.Namespace, selector.Name)
					continue
				}
				deploys = append(deploys, deploy)
			}
		}

		for _, deploy := range deploys {
			if util.IfPodMatchDeploy(deploy, pod) {
				return deploy, policy, nil
			}
		}
	}

	// find nothing
	return nil, nil, nil
}

func (ngs *NodeGroupScheduling) PreFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod) *framework.Status {
	deploy, policy, err := ngs.getRelativeDeployAndPolicy(pod)
	if err != nil {
		return framework.NewStatus(
			framework.Success,
			fmt.Sprintf("failed to get relative policy for pod %s/%s, %v, fall back to normal schedule",
				pod.Namespace, pod.Name, err),
		)
	}

	if policy == nil {
		// no relative policy
		// schedule this pod in normal way
		return framework.NewStatus(framework.Success, fmt.Sprintf("no relative policy find for pod %s/%s, fall back to normal schedule",
			pod.Namespace, pod.Name))
	}

	state.Write(NodeGroupKey, &nodeGroupStateData{
		policyNamespace: policy.Namespace,
		policyName:      policy.Name,
		policy:          policy,
		deploy:          deploy,
	})

	return framework.NewStatus(framework.Success, fmt.Sprintf("find policy %s/%s for pod %s/%s", policy.Namespace, policy.Name, pod.Namespace, pod.Name))
}

func (ngs *NodeGroupScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ngs *NodeGroupScheduling) desiredPodsNumInTargetNodeGroups(policy *groupmanagementv1alpha1.PropagationPolicy, deploy *appsv1.Deployment) map[string]int32 {
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

func (ngs *NodeGroupScheduling) currentPodsNumInTargetNodeGroups(deploy *appsv1.Deployment, policy *groupmanagementv1alpha1.PropagationPolicy) (map[string]int32, map[string]string, error) {
	// get all relative nodegroups
	targetNodeGroups := []*groupmanagementv1alpha1.NodeGroup{}
	for _, weight := range policy.Spec.Placement.StaticWeightList {
		for _, name := range weight.NodeGroupNames {
			// TODO:
			// make nodegroup non-namespaced
			nodegroup, err := ngs.nodgroupInformer.Lister().NodeGroups("default").Get(name)
			if err != nil {
				klog.Errorf("failed to get nodegroup %s, %v", name, err)
				continue
			}
			targetNodeGroups = append(targetNodeGroups, nodegroup)
		}
	}

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

	selector, err := util.GetPodSelectorFromDeploy(deploy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pod selector from deploy: %s/%s, %v", deploy.Namespace, deploy.Name, err)
	}
	pods, err := ngs.podInformer.Lister().List(selector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pods of deploy: %s/%s, %v", deploy.Namespace, deploy.Name, err)
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
	return currentPodsInTargetNodeGroups, nodeToNodeGroup, nil

}

func (ngs *NodeGroupScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	var policyData *nodeGroupStateData
	if data, err := state.Read(NodeGroupKey); err != nil {
		// fall back to normal pod scheduling
		return framework.NewStatus(framework.Success, "")
	} else if d, ok := data.(*nodeGroupStateData); !ok {
		// TODO:
		// Consider if we should fall back to normal scheduling.
		return framework.NewStatus(framework.Success, fmt.Sprintf("convert StateDate to nodeGroupStateData failed, fall back to normal schedule"))
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.desiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, nodeToNodeGroup, err := ngs.currentPodsNumInTargetNodeGroups(deploy, policy)
	if err != nil {
		return framework.NewStatus(framework.Success, fmt.Sprintf("CurrentPodsNumInTargetNodeGroups failed: %v, fall back to normal schedule", err))
	}

	nodegroup, find := nodeToNodeGroup[nodeInfo.Node().Name]
	if !find {
		// This node is not in any nodegroup
		return framework.NewStatus(framework.Unschedulable, "")
	}

	desciredPodsNum, currentPodsNum := desiredPodsNumOfEachNodeGroup[nodegroup], currentPodsNumOfEachNodeGroup[nodegroup]
	if currentPodsNum >= desciredPodsNum {
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("nodegroup %s already has enough pod num %d", nodegroup, currentPodsNum))
	}
	return framework.NewStatus(framework.Success, "")
}

func (ngs *NodeGroupScheduling) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodename string) (int64, *framework.Status) {
	var policyData *nodeGroupStateData
	if data, err := state.Read(NodeGroupKey); err != nil {
		// fall back to normal pod scheduling
		return 0, framework.NewStatus(framework.Success, "")
	} else if d, ok := data.(*nodeGroupStateData); !ok {
		return 0, framework.NewStatus(framework.Success, fmt.Sprintf("convert StateDate to nodeGroupStateData failed, fall back to normal schedule"))
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.desiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, nodesInNodeGroup, err := ngs.currentPodsNumInTargetNodeGroups(deploy, policy)
	if err != nil {
		return 0, framework.NewStatus(framework.Success, fmt.Sprintf("CurrentPodsNumInTargetNodeGroups failed: %v, fall back to normal schedule", err))
	}

	var diffList nodeGroupItemSlice
	for nodegroup, desiredNum := range desiredPodsNumOfEachNodeGroup {
		diffPodsNum := desiredNum - currentPodsNumOfEachNodeGroup[nodegroup]
		diffList = append(diffList, nodeGroupItem{nodeGroupName: nodegroup, podsNum: diffPodsNum})
	}

	diffList = sort.Reverse(diffList).(nodeGroupItemSlice)
	diffMap := make(map[string]nodeGroupItem, diffList.Len())
	for i := range diffList {
		diffList[i].rank = i + 1
		nodeGroupName := diffList[i].nodeGroupName
		diffMap[nodeGroupName] = diffList[i]
	}

	if nodegroup, ok := nodesInNodeGroup[nodename]; ok {
		ratio := (float64)(diffMap[nodegroup].rank) / (float64)(diffList.Len())
		score := (int64)(10 - 10*ratio)
		return score, framework.NewStatus(framework.Success, "")
	}
	return 0, framework.NewStatus(framework.Error,
		fmt.Sprintf("failed find node %s in any nodegroup, this should not happen in score plugin after filer running expectedly", nodename))
}

func (ngs *NodeGroupScheduling) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
