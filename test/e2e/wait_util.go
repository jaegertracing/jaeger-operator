package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/api/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// WaitForStatefulset checks to see if a given statefulset has the desired number of replicas available after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForStatefulset(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		statefulset, err := kubeclient.AppsV1().StatefulSets(namespace).Get(ctxWithTimeout, name, metav1.GetOptions{})
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
		t.Logf("Failed waiting for statefulset %s after %s\n", name, time.Since(start))
		return err
	}
	t.Logf("Statefulset %s available after %s\n", name, time.Since(start))
	return nil
}

// WaitForDaemonSet checks to see if a given daemonset has the desired number of instances available after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		daemonset, err := kubeclient.AppsV1().DaemonSets(namespace).Get(ctxWithTimeout, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}

		if daemonset.Status.NumberReady > 0 && daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s daemonsets (%d/%d)\n", name, daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
		return false, nil
	})
	if err != nil {
		t.Logf("Failed waiting for daemonset %s after %s\n", name, time.Since(start))
		return err
	}
	t.Logf("DaemonSet %s available after %s\n", name, time.Since(start))
	return nil
}

// WaitForIngress checks to see if a given ingress' load balancer is ready after a specified amount of time
// See #WaitForDeployment for the full semantics
func WaitForIngress(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) (*v1beta1.Ingress, error) {
	start := time.Now()
	var ingress *v1beta1.Ingress
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		ingress, err = kubeclient.NetworkingV1beta1().Ingresses(namespace).Get(ctxWithTimeout, name, metav1.GetOptions{})
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
		t.Logf("Failed waiting for ingress after %s\n", time.Since(start))
		return ingress, err
	}
	t.Logf("Ingress available after %s\n", time.Since(start))
	return ingress, nil
}

// WaitForJob checks to see if a given job has completed successfully
// See #WaitForDeployment for the full semantics
func WaitForJob(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		job, err := kubeclient.BatchV1().Jobs(namespace).Get(ctxWithTimeout, name, metav1.GetOptions{})
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
		t.Logf("Failed waiting for job %s after %s\n", name, time.Since(start))
		return err
	}
	t.Logf("Job %s succeeded after %s\n", name, time.Since(start))
	return nil
}

// WaitForJobOfAnOwner checks to see if a given job has completed successfully
// See #WaitForDeployment for the full semantics
func WaitForJobOfAnOwner(t *testing.T, kubeclient kubernetes.Interface, namespace, ownerName string, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		jobList, err := kubeclient.BatchV1().Jobs(namespace).List(ctxWithTimeout, metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s job owner\n", ownerName)
				return false, nil
			}
			return false, err
		}
		for _, j := range jobList.Items {
			for _, r := range j.OwnerReferences {
				if ownerName == r.Name && j.Status.Succeeded > 0 && j.Status.Failed == 0 && j.Status.Active == 0 {
					return true, nil
				}
			}
		}
		t.Logf("Waiting for job of owner %s to succeed.", ownerName)
		return false, nil
	})
	if err != nil {
		t.Logf("Failed waiting for job of an owner %s after %s\n", ownerName, time.Since(start))
		return err
	}
	t.Logf("Job of owner %s succeeded after %s\n", ownerName, time.Since(start))
	return nil
}

// WaitForCronJob checks to see if a given cron job scheduled a job
// See #WaitForDeployment for the full semantics
func WaitForCronJob(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cronJob, err := kubeclient.BatchV1beta1().CronJobs(namespace).Get(ctxWithTimeout, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s cronjob\n", name)
				return false, nil
			}
			return false, err
		}
		if cronJob.Status.LastScheduleTime != nil {
			return true, nil
		}
		t.Logf("Waiting for conjob %s to have scheduled", name)
		return false, nil
	})
	if err != nil {
		t.Logf("Failed waiting for cronjob %s after %s\n", name, time.Since(start))
		return err
	}
	t.Logf("CronJob %s succeeded after %s\n", name, time.Since(start))
	return nil
}

// WaitForDeployment waits for a deployment to finish and reports how long the operation took
func WaitForDeployment(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	start := time.Now()
	err := e2eutil.WaitForDeployment(t, fw.KubeClient, namespace, name, replicas, retryInterval, timeout)
	elapsed := time.Since(start)
	logrus.Infof("Deployment of %s in namespace %s took %s\n", name, namespace, elapsed)
	return err
}

// WaitForSecret waits for a secret to be available
func WaitForSecret(secretName, secretNamespace string) {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		secret, err := fw.KubeClient.CoreV1().Secrets(secretNamespace).Get(ctxWithTimeout, secretName, metav1.GetOptions{})
		if err == nil {
			logrus.Debugf("Found secret %s\n", secret.Name)
			return true, nil
		} else if err != nil && strings.Contains(err.Error(), "not found") {
			return false, nil
		} else {
			return false, err
		}
	})
	require.NoError(t, err)
}
