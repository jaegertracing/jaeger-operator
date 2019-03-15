package e2e

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DaemonSet runs a test with the agent as DaemonSet, but uses ingress to access it rather than portforwarding
func DaemonSetWithIngress(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := daemonsetTestWithIngress(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func daemonsetTestWithIngress(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	// This does not currently work on OpenShift as it creates a route, and the kubeclient call
	// in WaitForIngress doesn't find that.  We either need to figure out how to get kubeclient
	// to find routes (at the command line "kubectl get route.route.openshift.io" works) or use
	// the openshift client to find it.
	if isOpenShift(t, f) {
		t.Skipf("Test %s is not currently supported on OpenShift\n", t.Name())
	}

	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	j := getJaegerDefinition(namespace, "agent-as-daemonset")

	log.Infof("passing %v", j)
	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = WaitForDaemonSet(t, f.KubeClient, namespace, "agent-as-daemonset-agent-daemonset", retryInterval, timeout)
	if err != nil {
		return err
	}

	selector := map[string]string{"app": "vertx-create-span"}
	dep := getVertxDeployment(namespace, selector)
	err = f.Client.Create(goctx.TODO(), dep, cleanupOptions)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "vertx-create-span", 1, retryInterval, 3 * timeout)
	if err != nil {
		return err
	}

	err = WaitForIngress(t, f.KubeClient, namespace, "agent-as-daemonset-query", retryInterval, timeout)
	if err != nil {
		return err
	}

	i, err := f.KubeClient.ExtensionsV1beta1().Ingresses(namespace).Get("agent-as-daemonset-query", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if len(i.Status.LoadBalancer.Ingress) != 1 {
		return fmt.Errorf("Wrong number of ingresses. Expected 1, was %v", len(i.Status.LoadBalancer.Ingress))
	}

	address := i.Status.LoadBalancer.Ingress[0].IP
	url := fmt.Sprintf("http://%s/api/traces?service=order", address)
	c := http.Client{Timeout: time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := c.Do(req)
		if err != nil {
			return false, err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		resp := &resp{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, err
		}

		return len(resp.Data) > 0, nil
	})
}
