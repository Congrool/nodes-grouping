package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

func IfPodMatchDeploy(deploy *appsv1.Deployment, pod *corev1.Pod) bool {
	selector, err := GetPodSelectorFromDeploy(deploy)
	if err != nil {
		klog.Errorf("failed to get selector from labelselector, err: %s")
		return false
	}

	podLabels := pod.Labels
	if selector.Matches(labels.Set(podLabels)) {
		return true
	}
	return false
}

func GetPodSelectorFromDeploy(deploy *appsv1.Deployment) (labels.Selector, error) {
	labelSelector := deploy.Spec.Selector
	return metav1.LabelSelectorAsSelector(labelSelector)
}
