package e2e

import (
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// WaitForStatefulset checks to see if a given statefulset has the desired number of replicas available after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForStatefulset(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		statefulset, err := kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s statefulset\n", name)
				return false, nil
			}
			return false, err
		}

		if statefulset.Status.ReadyReplicas == statefulset.Status.CurrentReplicas {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, statefulset.Status.ReadyReplicas, statefulset.Status.CurrentReplicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Statefulset available\n")
	return nil
}
