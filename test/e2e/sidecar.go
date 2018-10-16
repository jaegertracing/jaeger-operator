package e2e

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Sidecar runs a test with the agent as sidecar
func Sidecar(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := sidecarTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func sidecarTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	j := &v1alpha1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "io.jaegertracing/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-as-sidecar",
			Namespace: namespace,
		},
		Spec: v1alpha1.JaegerSpec{
			Strategy: "all-in-one",
			AllInOne: v1alpha1.JaegerAllInOneSpec{},
			Agent: v1alpha1.JaegerAgentSpec{
				Options: v1alpha1.NewOptions(map[string]interface{}{
					"log-level": "debug",
				}),
			},
		},
	}

	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	selector := map[string]string{"app": "vertx-create-span-sidecar"}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "vertx-create-span-sidecar",
			Namespace:   namespace,
			Annotations: map[string]string{inject.Annotation: "true"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image: "jaegertracing/vertx-create-span:operator-e2e-tests",
						Name:  "vertx-create-span-sidecar",
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 8080,
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 1,
						},
						LivenessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 1,
						},
					}},
				},
			},
		},
	}
	err = f.Client.Create(goctx.TODO(), dep, cleanupOptions)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "vertx-create-span-sidecar", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = WaitForIngress(t, f.KubeClient, namespace, "agent-as-sidecar-query", retryInterval, timeout)
	if err != nil {
		return err
	}

	i, err := f.KubeClient.ExtensionsV1beta1().Ingresses(namespace).Get("agent-as-sidecar-query", metav1.GetOptions{})
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
