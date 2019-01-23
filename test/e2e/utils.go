package e2e

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPod returns pod name
func GetPod(namespace, namePrefix, containsImage string, kubeclient kubernetes.Interface) (v1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return v1.Pod{}, err
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
	return v1.Pod{}, errors.New(fmt.Sprintf("could not find pod with image %s", containsImage))
}
