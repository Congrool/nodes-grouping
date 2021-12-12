package nodegroup

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	groupv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/group/v1alpha1"
)

const (
	// ControllerName is the controller name that will be used when reporting events.
	ControllerName = "nodegroup-controller"
	// MonitorRetrySleepTime is the amount of time the node group controller that should
	// sleep between retrying NodeGroup updates.
	MonitorRetrySleepTime = 20 * time.Millisecond
)

// Controller is to sync NodeGroup.
type Controller struct {
	client.Client

	// TODO:
	// not used
	EventRecorder record.EventRecorder
}

// Reconcile performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	klog.Infof("Reconciling nodeGroup %s", req.NamespacedName.Name)

	nodeGroup := &groupv1alpha1.NodeGroup{}
	if err := c.Client.Get(context.TODO(), req.NamespacedName, nodeGroup); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}

		return controllerruntime.Result{Requeue: true}, err
	}

	if !nodeGroup.DeletionTimestamp.IsZero() {
		return c.removeNodeGroup(nodeGroup)
	}

	return c.syncNodeGroup(nodeGroup)
}

func (c *Controller) syncNodeGroup(nodeGroup *groupv1alpha1.NodeGroup) (controllerruntime.Result, error) {
	matchLables := nodeGroup.Spec.MatchLabels

	nodeList, err := c.GetNodesByLabelSelector(labels.SelectorFromSet(labels.Set(matchLables)))
	if err != nil {
		klog.Errorf("Error while get nodes by labels %v. err: ", matchLables, err)
		return controllerruntime.Result{Requeue: true}, err
	}

	var containedNodes []string
	for k := range nodeList.Items {
		containedNodes = append(containedNodes, nodeList.Items[k].Name)
	}

	if !equality.Semantic.DeepEqual(nodeGroup.Status.ContainedNodes, containedNodes) {
		nodeGroup.Status.ContainedNodes = containedNodes
		c.Status().Update(context.TODO(), nodeGroup)
	}

	return controllerruntime.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	return utilerrors.NewAggregate([]error{
		controllerruntime.NewControllerManagedBy(mgr).For(&groupv1alpha1.NodeGroup{}).Complete(c),
	})
}

func (c *Controller) removeNodeGroup(nodeGroup *groupv1alpha1.NodeGroup) (controllerruntime.Result, error) {
	if err := c.Client.Delete(context.TODO(), nodeGroup); err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Error while deleting nodegroup %s: %s", nodeGroup)
		return controllerruntime.Result{Requeue: true}, err
	}

	return controllerruntime.Result{}, nil
}

// GetNodesByLabelSelector get NodeList by matching label selector
func (c *Controller) GetNodesByLabelSelector(selector labels.Selector) (*corev1.NodeList, error) {
	nodeList := &corev1.NodeList{}
	err := c.Client.List(context.TODO(), nodeList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	return nodeList, nil
}
