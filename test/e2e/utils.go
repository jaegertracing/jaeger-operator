package e2e

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPod returns pod name
func GetPod(namespace, namePrefix, ownerNamePrefix string, kubeclient kubernetes.Interface) (v1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{IncludeUninitialized:true})
	if err != nil {
		return v1.Pod{}, err
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, namePrefix) {
			for _, or := range pod.OwnerReferences {
				if strings.HasPrefix(or.Name, ownerNamePrefix) {
					return pod, nil
				}
			}
		}
	}
	return v1.Pod{}, errors.New(fmt.Sprintf("could not find pod of owner %s", ownerNamePrefix))
}
