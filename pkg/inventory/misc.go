package inventory

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initK8sObjectMeta will set the required default settings to
// kubernetes objects metadata if is required.
func initK8sObjectMeta(obj metav1.Object) {
	if obj.GetLabels() == nil {
		obj.SetLabels(map[string]string{})
	}

	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
}
