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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// DaemonSet runs a test with the agent as DaemonSet
func DaemonSet(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := daemonsetTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func daemonsetTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	// TODO restore this after fix of https://github.com/jaegertracing/jaeger-operator/issues/322
	if isOpenShift(t, f) {
		t.Skipf("Test %s is not currently supported on OpenShift because of issue 322\n", t.Name())
	}
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	j := jaegerAgentAsDaemonsetDefinition(namespace, "agent-as-daemonset")

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
		fmt.Println("Failed waiting for vertx deployment")
		return err
	}

	queryPod, err := GetPod(namespace, "agent-as-daemonset", "jaegertracing/all-in-one", f.KubeClient)
	if err != nil {
		fmt.Println("Failed waiting for pod agent-as-daemonset")
		return err
	}

	portForw, closeChan, err := CreatePortForward(namespace, queryPod.Name, []string{"16686"}, f.KubeConfig)
	if err != nil {
		fmt.Println("Failed waiting for port forward")
		return err
	}
	defer portForw.Close()
	defer close(closeChan)

	url := "http://localhost:16686/api/traces?service=order"
	c := http.Client{Timeout: time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Failed waiting for get on " + url)
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

func getVertxDeployment(namespace string, selector map[string]string) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vertx-create-span",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "jaegertracing/vertx-create-span:operator-e2e-tests",
						Name:  "vertx-create-span",
						Env: []corev1.EnvVar{
							corev1.EnvVar{
								Name: "JAEGER_AGENT_HOST",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.hostIP",
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 8080,
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(8080),
								},
							},
							InitialDelaySeconds: 1,
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
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
	return dep
}

func jaegerAgentAsDaemonsetDefinition(namespace string, name string) *v1.Jaeger {
	j := &v1.Jaeger{
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
			AllInOne: v1.JaegerAllInOneSpec{},
			Agent: v1.JaegerAgentSpec{
				Strategy: "DaemonSet",
				Options: v1.NewOptions(map[string]interface{}{
					"log-level": "debug",
				}),
			},
		},
	}
	return j
}
