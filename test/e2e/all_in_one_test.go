package e2e

import (
	goctx "context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	osv1 "github.com/openshift/api/route/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

const TrackingID = "MyTrackingId"

func JaegerAllInOne(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := allInOneTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err := allInOneWithIngressTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err := allInOneWithUIConfigTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func allInOneTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// create jaeger custom resource
	exampleJaeger := getJaegerAllInOneDefinition(namespace, "my-jaeger")

	log.Infof("passing %v", exampleJaeger)
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "my-jaeger", 1, retryInterval, timeout)
}

func allInOneWithIngressTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	// create jaeger custom resource
	ingressEnabled := true
	name := "my-jaeger-with-ingress"
	exampleJaeger := getJaegerAllInOneDefinition(namespace, name)
	exampleJaeger.Spec.Ingress = v1.JaegerIngressSpec{
		Enabled:  &ingressEnabled,
		Security: v1.IngressSecurityNoneExplicit,
	}

	log.Infof("passing %v", exampleJaeger)
	err = f.Client.Create(goctx.TODO(), exampleJaeger, &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval})
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, name, 1, retryInterval, 3*timeout)
	if err != nil {
		t.Errorf("Error waiting for deployment of %s: %v\n", name, err)
		return err
	}

	var url string
	var httpClient http.Client
	if isOpenShift(t) {
		route, err := findRoute(t, f, name)
		if err != nil {
			return err
		}

		if len(route.Status.Ingress) != 1 {
			return fmt.Errorf("Wrong number of ingresses. Expected 1, was %v", len(route.Status.Ingress))
		}

		url = fmt.Sprintf("https://%s/api/services", route.Spec.Host)
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = http.Client{Timeout: 30 * time.Second, Transport: transport}
	} else {
		ingress, err := WaitForIngress(t, f.KubeClient, namespace, "my-jaeger-with-ingress-query", retryInterval, timeout)
		if err != nil {
			return err
		}

		if len(ingress.Status.LoadBalancer.Ingress) != 1 {
			return fmt.Errorf("Wrong number of ingresses. Expected 1, was %v", len(ingress.Status.LoadBalancer.Ingress))
		}

		address := ingress.Status.LoadBalancer.Ingress[0].IP
		url = fmt.Sprintf("http://%s/api/services", address)
		httpClient = http.Client{Timeout: time.Second}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// Hit this url once to make Jaeger itself create a trace, then it will show up in services
	httpClient.Do(req)

	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		res, err := httpClient.Do(req)
		if err != nil {
			return false, err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		resp := &services{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, nil
		}

		for _, v := range resp.Data {
			if v == "jaeger-query" {
				return true, nil
			}
		}

		return false, nil
	})
}

func allInOneWithUIConfigTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	basePath := "/jaeger"

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "all-in-one-with-ui-config",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1.JaegerAllInOneSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"query.base-path": basePath,
				}),
			},
			UI: v1.JaegerUISpec{
				Options: v1.NewFreeForm(map[string]interface{}{
					"tracking": map[string]interface{}{
						"gaID": TrackingID,
					},
				}),
			},
		},
	}

	j.Spec.Annotations = map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "false",
	}

	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "all-in-one-with-ui-config", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	queryPod, err := GetPod(namespace, "all-in-one-with-ui-config", "jaegertracing/all-in-one", f.KubeClient)
	if err != nil {
		return err
	}

	portForward, closeChan, err := CreatePortForward(namespace, queryPod.Name, []string{"16686"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer portForward.Close()
	defer close(closeChan)

	url := fmt.Sprintf("http://%s:16686/%s/search", kubernetesHost, basePath)
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

		if res.StatusCode != 200 {
			return false, fmt.Errorf("unexpected status code %d", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return false, err
		}

		if len(body) == 0 {
			return false, fmt.Errorf("empty body")
		}

		if !strings.Contains(string(body), TrackingID) {
			return false, fmt.Errorf("body does not include tracking id: %s", TrackingID)
		}

		return true, nil
	})
}

func getJaegerAllInOneDefinition(namespace string, name string) *v1.Jaeger {
	exampleJaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			AllInOne: v1.JaegerAllInOneSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"log-level":         "debug",
					"memory.max-traces": 10000,
				}),
			},
		},
	}
	return exampleJaeger
}

func findRoute(t *testing.T, f *framework.Framework, name string) (*osv1.Route, error) {
	routeList := &osv1.RouteList{}
	err := wait.Poll(retryInterval, timeout, func() (bool, error) {
		opts := &client.ListOptions{}
		if err := f.Client.List(context.Background(), opts, routeList); err != nil {
			return false, err
		}
		if len(routeList.Items) >= 1 {
			return true, nil
		} else {
			return false, nil
		}
	})

	if err != nil {
		t.Errorf("Failed waiting for route: %v", err)
		return nil, err
	}

	for _, r := range routeList.Items {
		if strings.HasPrefix(r.Spec.Host, name) {
			return &r, nil
		}
	}
	return nil, errors.New("Could not find route")
}
