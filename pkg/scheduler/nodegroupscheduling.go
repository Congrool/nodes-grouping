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
	klog.V(2).Info("start to init NodeGroupScheduling plugin")
	args, ok := obj.(*config.NodeGroupSchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("NodeGroupScheduling wants args to be of type NodeGroupSchedulingArgs, got %T, %v", obj, obj)
	}

	conf, err := clientcmd.BuildConfigFromFlags("", *args.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("NodeGroupScheduling: failed to init rest.Config: %v", err)
	}

	clientset := groupmanagementclientset.NewForConfigOrDie(conf)
	informerFactory := groupinformerfactory.NewSharedInformerFactory(clientset, 0)
	policyInformer := informerFactory.Groupmanagement().V1alpha1().PropagationPolicies()
	nodegroupInformer := informerFactory.Groupmanagement().V1alpha1().NodeGroups()
	podInformer := handle.SharedInformerFactory().Core().V1().Pods()
	deployInformer := handle.SharedInformerFactory().Apps().V1().Deployments()

	groupManager := manager.New(policyInformer, nodegroupInformer, deployInformer, podInformer)

	hasSyncFuncs := []cache.InformerSynced{
		podInformer.Informer().HasSynced,
		deployInformer.Informer().HasSynced,
		policyInformer.Informer().HasSynced,
		nodegroupInformer.Informer().HasSynced,
	}

	ctx := context.TODO()
	informerFactory.Start(ctx.Done())
	// TODO: run informer in the handle.SharedInformerFactory
	go podInformer.Informer().Run(ctx.Done())
	go deployInformer.Informer().Run(ctx.Done())

	klog.V(2).Info("NodeGroupScheduling: waitting for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), hasSyncFuncs...) {
		err := fmt.Errorf("WaitForCacheSync failed")
		klog.ErrorS(err, "NodeGroupScheduling cannot sync caches")
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
		return framework.NewStatus(framework.Success,
			fmt.Sprintf("failed to get relative PropagationPolicy for pod %s/%s, %v, fall back to normal scheduling process", pod.Namespace, pod.Name, err))
	}

	if policy == nil {
		// no relative policy found
		// schedule this pod in normal way
		return framework.NewStatus(framework.Success,
			fmt.Sprintf("no relative PropagationPolicy find for pod %s/%s, fall back to normal scheduling process", pod.Namespace, pod.Name))
	}

	klog.V(2).Infof("NodeGroupScheduling PreFilter: get relative deploy %s/%s and PropagationPolicy %s/%s of pod %s/%s",
		deploy.Namespace, deploy.Name, policy.Namespace, policy.Name, pod.Namespace, pod.Name)

	state.Write(NodeGroupKey, &nodeGroupStateData{
		policy: policy,
		deploy: deploy,
	})

	return framework.NewStatus(framework.Success, fmt.Sprintf("find PropagationPolicy %s/%s for pod %s/%s", policy.Namespace, policy.Name, pod.Namespace, pod.Name))
}

func (ngs *NodeGroupScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ngs *NodeGroupScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if !hasNodeGroupStateData(state) {
		klog.V(2).Infof("pod %s/%s is not controlled by and propagation policy, fall back to noraml scheduling process", pod.Namespace, pod.Name)
		return framework.NewStatus(framework.Success, fmt.Sprintf("pod %s/%s is not controlled by any propagation policy", pod.Namespace, pod.Name))
	}

	stateData, err := retrieveNodeGroupStateData(state)
	if err != nil {
		err = fmt.Errorf("failed to retrieve NodeGroupStateData, %v", err)
		klog.ErrorS(err, "NodeGroupScheduling Filter error")
		return framework.NewStatus(framework.Error, err.Error())
	}

	policy, deploy := stateData.policy, stateData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.groupManager.DesiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, err := ngs.groupManager.CurrentPodsNumInTargetNodeGroups(policy, deploy)
	if err != nil {
		err = fmt.Errorf("failed to count current pods number in target nodegroups of policy: %s/%s, %v", policy.Namespace, policy.Name, err)
		klog.ErrorS(err, "NodeGroupScheduling Filter error")
		return framework.NewStatus(framework.Error, err.Error())
	}

	nodeToNodeGroup := ngs.groupManager.MapNodeToNodeGroup(policy)
	nodegroup, find := nodeToNodeGroup[nodeInfo.Node().Name]
	if !find {
		// This node is not in any target nodegroup
		klog.V(2).Infof("NodeGroupScheduling Filter: node %s is not in any nodegroup which pod %s/%s requires in policy %s/%s, mark it as unschedulable",
			nodeInfo.Node().Name, pod.Namespace, pod.Name, policy.Namespace, policy.Name)
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("node %s is not what pod requires in policy %s/%s", nodeInfo.Node().Name, policy.Namespace, policy.Name))
	}

	desciredPodsNum, currentPodsNum := desiredPodsNumOfEachNodeGroup[nodegroup], currentPodsNumOfEachNodeGroup[nodegroup]
	if currentPodsNum >= desciredPodsNum {
		klog.V(2).Infof("NodeGroupScheduling Filter: nodegroup %s which node %s belongs to has already had enough pod, do not schedule the pod %s/%s to the node",
			nodegroup, nodeInfo.Node().Name, pod.Namespace, pod.Name)
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("nodegroup %s has already had enough pod num %d", nodegroup, currentPodsNum))
	}
	klog.V(2).Infof("NodeGroupScheduling Filter: pod %s/%s can be scheduled to node %s", pod.Namespace, pod.Name, nodeInfo.Node().Name)
	return framework.NewStatus(framework.Success, "")
}

func (ngs *NodeGroupScheduling) Score(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodename string) (int64, *framework.Status) {
	if !hasNodeGroupStateData(state) {
		klog.V(2).Infof("pod %s/%s is not controlled by any propagation policy, fall back to normal scheduling process", pod.Namespace, pod.Name)
		return 0, framework.NewStatus(framework.Success, fmt.Sprintf("pod %s/%s is not controlled by any propagation policy", pod.Namespace, pod.Name))
	}

	// start to process pod controlled by some propagation policy
	stateData, err := retrieveNodeGroupStateData(state)
	if err != nil {
		err = fmt.Errorf("failed to retrieve NodeGroupStateData, %v", err)
		klog.ErrorS(err, "NodeGroupScheduling Score error")
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	policy, deploy := stateData.policy, stateData.deploy
	desiredPodsNumOfEachNodeGroup := ngs.groupManager.DesiredPodsNumInTargetNodeGroups(policy, deploy)
	currentPodsNumOfEachNodeGroup, err := ngs.groupManager.CurrentPodsNumInTargetNodeGroups(policy, deploy)
	if err != nil {
		err = fmt.Errorf("failed to count current pods number in target nodegroups of policy: %s/%s, %v", policy.Namespace, policy.Name, err)
		klog.ErrorS(err, "NodeGroupScheduling Score error")
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	// sort all the relative nodegroups based on the difference between desired pod number and current pod number
	//
	// Note:
	// For each nodegroup here, we have current pod number < want pod number. Otherwise, it will be filtered out
	// by the filter.
	var diffList nodeGroupItemSlice
	for nodegroup, desiredNum := range desiredPodsNumOfEachNodeGroup {
		diffPodsNum := desiredNum - currentPodsNumOfEachNodeGroup[nodegroup]
		diffList = append(diffList, nodeGroupItem{nodeGroupName: nodegroup, podsNum: diffPodsNum})
	}
	sort.Sort(sort.Reverse(diffList))
	diffMap := make(map[string]nodeGroupItem, diffList.Len())
	for i := range diffList {
		diffList[i].rank = i + 1
		nodeGroupName := diffList[i].nodeGroupName
		diffMap[nodeGroupName] = diffList[i]
	}

	// score the node based on the rank of the nodegroup it belongs to.
	nodeToNodeGroup := ngs.groupManager.MapNodeToNodeGroup(policy)
	if nodegroup, ok := nodeToNodeGroup[nodename]; ok {
		ratio := (float64)(diffMap[nodegroup].rank) / (float64)(diffList.Len())
		score := (int64)(10 - 10*ratio)
		klog.V(2).Infof("node %s gets score of %d for pod %s/%s", nodename, score, pod.Namespace, pod.Name)
		return score, framework.NewStatus(framework.Success, fmt.Sprintf("node %s gets score of %d for pod %s/%s", nodename, score, pod.Namespace, pod.Name))
	}
	err = fmt.Errorf("failed find node %s in any nodegroup, this should not happen in score plugin after filer running as expected", nodename)
	klog.ErrorS(err, "NodeGroupScheduling Score error")
	return 0, framework.NewStatus(framework.Error, err.Error())
}

func (ngs *NodeGroupScheduling) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func hasNodeGroupStateData(state *framework.CycleState) bool {
	_, err := state.Read(NodeGroupKey)
	return err == nil
}

func retrieveNodeGroupStateData(state *framework.CycleState) (*nodeGroupStateData, error) {
	if data, err := state.Read(NodeGroupKey); err != nil {
		return nil, err
	} else {
		d, ok := data.(*nodeGroupStateData)
		if !ok {
			return nil, fmt.Errorf("failed to convert StateKey want type: nodeGroupStateData, got: %T", data)
		}
		// klog.V(2).Infof("NodeGroupScheduling Filter: get StateKey of pod %s/%s, deploy: %s/%s, policy: %s/%s", pod.Namespace, pod.Name, d.deploy.Namespace, d.deploy.Name, d.policy.Namespace, d.policy.Name)
		return d, nil
	}
}
