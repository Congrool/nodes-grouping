package scheduler

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	config "github.com/Congrool/nodes-grouping/pkg/apis/config"
	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	groupmanagementclientset "github.com/Congrool/nodes-grouping/pkg/generated/clientset/versioned"
	groupinformerfactory "github.com/Congrool/nodes-grouping/pkg/generated/informers/externalversions"
	"github.com/Congrool/nodes-grouping/pkg/scheduler/manager"
)

type nodeGroupStateData struct {
	policy *groupmanagementv1alpha1.PropagationPolicy
	deploy *appsv1.Deployment
}

func (nsd *nodeGroupStateData) Clone() framework.StateData {
	return &nodeGroupStateData{
		policy: nsd.policy,
		deploy: nsd.deploy,
	}
}

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
	groupManager manager.GroupManager
}

const (
	Name         = "NodeGroupScheduling"
	NodeGroupKey = "nodegroup"
)

var _ framework.PreFilterPlugin = &NodeGroupScheduling{}
var _ framework.FilterPlugin = &NodeGroupScheduling{}
var _ framework.ScorePlugin = &NodeGroupScheduling{}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.NodeGroupSchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("NodeGroupScheduling wants args to be of type NodeGroupSchedulingArgs, got %T, %v", obj, obj)
	}

	conf, err := clientcmd.BuildConfigFromFlags("", *args.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init rest.Config: %v", err)
	}

	clientset := groupmanagementclientset.NewForConfigOrDie(conf)
	informerFactory := groupinformerfactory.NewSharedInformerFactory(clientset, 0)
	policyInformer := informerFactory.Groupmanagement().V1alpha1().PropagationPolicies()
	nodegroupInformer := informerFactory.Groupmanagement().V1alpha1().NodeGroups()
	podInformer := handle.SharedInformerFactory().Core().V1().Pods()
	deployInformer := handle.SharedInformerFactory().Apps().V1().Deployments()

	groupManager := manager.New(policyInformer, nodegroupInformer, deployInformer, podInformer)

	ctx := context.TODO()
	informerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), policyInformer.Informer().HasSynced, nodegroupInformer.Informer().HasSynced) {
		err := fmt.Errorf("WaitForCacheSync failed")
		klog.ErrorS(err, "Cannot sync caches")
		return nil, err
	}

	return &NodeGroupScheduling{
		groupManager: groupManager,
	}, nil
}

func (ngs *NodeGroupScheduling) Name() string {
	return Name
}

func (ngs *NodeGroupScheduling) PreFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod) *framework.Status {
	policy, deploy, err := ngs.groupManager.GetRelativeDeployAndPolicy(pod)
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
		policy: policy,
		deploy: deploy,
	})

	return framework.NewStatus(framework.Success, fmt.Sprintf("find policy %s/%s for pod %s/%s", policy.Namespace, policy.Name, pod.Namespace, pod.Name))
}

func (ngs *NodeGroupScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ngs *NodeGroupScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	var policyData *nodeGroupStateData
	if data, err := state.Read(NodeGroupKey); err != nil {
		// fall back to normal pod scheduling
		return framework.NewStatus(framework.Success, "")
	} else if d, ok := data.(*nodeGroupStateData); !ok {
		// TODO:
		// Consider if we should fall back to normal scheduling.
		return framework.NewStatus(framework.Success, "convert StateDate to nodeGroupStateData failed, fall back to normal schedule")
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.groupManager.DesiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, err := ngs.groupManager.CurrentPodsNumInTargetNodeGroups(policy, deploy)
	if err != nil {
		return framework.NewStatus(framework.Success, fmt.Sprintf("CurrentPodsNumInTargetNodeGroups failed: %v, fall back to normal schedule", err))
	}

	nodeToNodeGroup := ngs.groupManager.MapNodeToNodeGroup(policy)
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
		return 0, framework.NewStatus(framework.Success, "convert StateDate to nodeGroupStateData failed, fall back to normal schedule")
	} else {
		policyData = d
	}

	policy, deploy := policyData.policy, policyData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.groupManager.DesiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, err := ngs.groupManager.CurrentPodsNumInTargetNodeGroups(policy, deploy)
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

	nodeToNodeGroup := ngs.groupManager.MapNodeToNodeGroup(policy)
	if nodegroup, ok := nodeToNodeGroup[nodename]; ok {
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
