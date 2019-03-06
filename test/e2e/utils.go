package e2e

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPod returns pod name
func GetPod(namespace, namePrefix, containsImage string, kubeclient kubernetes.Interface) (corev1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return corev1.Pod{}, err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, namePrefix) {
			for _, c := range pod.Spec.Containers {
				if strings.Contains(c.Image, containsImage) {
					return pod, nil
				}
			}
		}
	}
	return corev1.Pod{}, fmt.Errorf("could not find pod with image %s", containsImage)
}
