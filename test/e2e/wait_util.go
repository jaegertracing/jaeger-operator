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
		t.Logf("Waiting for full availability of %s statefulsets (%d/%d)\n", name, statefulset.Status.ReadyReplicas, statefulset.Status.CurrentReplicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Statefulset available\n")
	return nil
}

// WaitForDaemonSet checks to see if a given daemonset has the desired number of instances available after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		daemonset, err := kubeclient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}

		if daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s daemonsets (%d/%d)\n", name, daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("DaemonSet available\n")
	return nil
}

// WaitForIngress checks to see if a given ingress' load balancer is ready after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForIngress(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ingress, err := kubeclient.ExtensionsV1beta1().Ingresses(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s ingress\n", name)
				return false, nil
			}
			return false, err
		}

		if len(ingress.Status.LoadBalancer.Ingress) > 0 {
			return true, nil
		}
		t.Logf("Waiting for full availability of the ingress %s\n", name)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Ingress available\n")
	return nil
}

// WaitForJob checks to see if a given job has the completed successfuly
// See #WaitForDeployment for the full semantics
func WaitForJob(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		job, err := kubeclient.BatchV1().Jobs(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s job\n", name)
				return false, nil
			}
			return false, err
		}

		if job.Status.Succeeded > 0 && job.Status.Failed == 0 && job.Status.Active == 0 {
			return true, nil
		}
		t.Logf("Waiting for job %s to succeed. Succeeded: %d, failed: %d, active: %d\n", name, job.Status.Succeeded, job.Status.Failed, job.Status.Active)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Jobs succeeded\n")
	return nil
}
