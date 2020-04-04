package autodetect

import (
	"context"
	"fmt"
	"testing"
	"time"

	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	fakeRest "k8s.io/client-go/rest/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

func TestStart(t *testing.T) {
	defer viper.Reset()

	// sanity check
	assert.False(t, viper.IsSet("auth-delegator-available"))

	// prepare
	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	done := make(chan bool)
	go func() {
		for {
			if viper.IsSet("auth-delegator-available") {
				break
			}
			// it would typically take less than 10ms to get the first result already, so, it should wait only once
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// test
	b.Start()

	// verify
	select {
	case <-done:
		assert.True(t, viper.GetBool("auth-delegator-available"))
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timed out waiting for the start process to detect the capabilities")
	}
}

func TestStartContinuesInBackground(t *testing.T) {
	defer viper.Reset()

	// prepare
	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b := WithClients(cl, dcl, cl)

	done := make(chan bool)
	go func() {
		for {
			if viper.IsSet("auth-delegator-available") {
				break
			}
			// it would typically take less than 10ms to get the first result already, so, it should wait only once
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	b.Start()

	select {
	case <-done:
		assert.False(t, viper.GetBool("auth-delegator-available"))
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timed out waiting for the start process to detect the capabilities")
	}

	// test
	cl.CreateFunc = cl.Client.Create // triggers a change in the availability

	go func() {
		for {
			if viper.GetBool("auth-delegator-available") {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		done <- true
	}()

	// verify
	select {
	case <-done:
		assert.True(t, viper.GetBool("auth-delegator-available"))
	case <-time.After(6 * time.Second): // this one might take up to 5 seconds to run again + processing time
		assert.Fail(t, "timed out waiting for the start process to detect the new capabilities")
	}

}

func TestAutoDetectWithError(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// sanity check
	assert.False(t, viper.IsSet("platform"))
	assert.False(t, viper.IsSet("es-provision"))

	// set the error
	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return nil, fmt.Errorf("faked error")
	}

	// Check initial value of "platform"
	assert.Equal(t, "", viper.GetString("platform"))

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, "", viper.GetString("platform"))
	assert.False(t, viper.GetBool("es-provision"))
}

func TestAutoDetectOpenShift(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformAutoDetect)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{
			Groups: []metav1.APIGroup{{
				Name: "route.openshift.io",
			}},
		}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformOpenShift, viper.GetString("platform"))

	// set the error
	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return nil, fmt.Errorf("faked error")
	}

	// run autodetect again with failure
	b.autoDetectCapabilities()

	// verify again
	assert.Equal(t, v1.FlagPlatformOpenShift, viper.GetString("platform"))
}

func TestAutoDetectKubernetes(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformAutoDetect)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformKubernetes, viper.GetString("platform"))
}

func TestExplicitPlatform(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformOpenShift, viper.GetString("platform"))
}

func TestAutoDetectEsProvisionNoEsOperator(t *testing.T) {
	// prepare
	viper.Set("es-provision", v1.FlagProvisionElasticsearchAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionElasticsearchNo, viper.GetString("es-provision"))
}

func TestAutoDetectEsProvisionWithEsOperator(t *testing.T) {
	// prepare
	viper.Set("es-provision", v1.FlagProvisionElasticsearchAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{
			Groups: []metav1.APIGroup{{
				Name: "logging.openshift.io",
			}},
		}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionElasticsearchYes, viper.GetString("es-provision"))
}

func TestAutoDetectKafkaProvisionNoKafkaOperator(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaNo, viper.GetString("kafka-provision"))
}

func TestAutoDetectKafkaProvisionWithKafkaOperator(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{
			Groups: []metav1.APIGroup{{
				Name: "kafka.strimzi.io",
			}},
		}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaYes, viper.GetString("kafka-provision"))
}

func TestAutoDetectKafkaExplicitYes(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaYes)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaYes, viper.GetString("kafka-provision"))
}

func TestAutoDetectKafkaExplicitNo(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaNo)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaNo, viper.GetString("kafka-provision"))
}

func TestAutoDetectKafkaDefaultNoOperator(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaNo, viper.GetString("kafka-provision"))
}

func TestAutoDetectKafkaDefaultWithOperator(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{
			Groups: []metav1.APIGroup{{
				Name: "kafka.strimzi.io",
			}},
		}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaYes, viper.GetString("kafka-provision"))
}

func TestNoAuthDelegatorAvailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b := WithClients(cl, dcl, cl)

	// test
	b.detectClusterRoles(context.Background())

	// verify
	assert.False(t, viper.GetBool("auth-delegator-available"))
}

func TestAuthDelegatorBecomesAvailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b := WithClients(cl, dcl, cl)

	// test
	b.detectClusterRoles(context.Background())
	assert.False(t, viper.GetBool("auth-delegator-available"))

	cl.CreateFunc = cl.Client.Create
	b.detectClusterRoles(context.Background())
	assert.True(t, viper.GetBool("auth-delegator-available"))
}

func TestAuthDelegatorBecomesUnavailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.detectClusterRoles(context.Background())
	assert.True(t, viper.GetBool("auth-delegator-available"))

	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b.detectClusterRoles(context.Background())
	assert.False(t, viper.GetBool("auth-delegator-available"))
}

func TestCleanDeployments(t *testing.T) {
	for _, tt := range []struct {
		cap             string // caption for the test
		watchNamespace  string // the value for WATCH_NAMESPACE
		jaegerNamespace string // in which namespace the jaeger exists, empty for non existing
		deleted         bool   // whether the sidecar should have been deleted
	}{
		{
			cap:             "existing-same-namespace",
			watchNamespace:  "observability",
			jaegerNamespace: "observability",
			deleted:         false,
		},
		{
			cap:             "not-existing-same-namespace",
			watchNamespace:  "observability",
			jaegerNamespace: "",
			deleted:         true,
		},
		{
			cap:             "existing-watched-namespace",
			watchNamespace:  "observability,other-observability",
			jaegerNamespace: "other-observability",
			deleted:         false,
		},
		{
			cap:             "existing-non-watched-namespace",
			watchNamespace:  "observability",
			jaegerNamespace: "other-observability",
			deleted:         true,
		},
		{
			cap:             "existing-watching-all-namespaces",
			watchNamespace:  v1.WatchAllNamespaces,
			jaegerNamespace: "other-observability",
			deleted:         false,
		},
	} {
		t.Run(tt.cap, func(t *testing.T) {
			// prepare the test data
			viper.Set(v1.ConfigWatchNamespace, tt.watchNamespace)
			defer viper.Reset()

			jaeger := v1.NewJaeger(types.NamespacedName{
				Name:      "my-instance",
				Namespace: "observability", // at first, it exists in the same namespace as the deployment
			})

			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "mydep",
					Namespace:   "observability",
					Annotations: map[string]string{inject.Annotation: jaeger.Name},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "C1",
									Image: "image1",
								},
							},
						},
					},
				},
			}
			dep = inject.Sidecar(jaeger, dep)

			// sanity check
			require.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))

			// prepare the list of existing objects
			objs := []runtime.Object{dep}
			if len(tt.jaegerNamespace) > 0 {
				jaeger.Namespace = tt.jaegerNamespace // now, it exists only in this namespace
				objs = append(objs, jaeger)
			}

			// prepare the client
			s := scheme.Scheme
			s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
			s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
			cl := fake.NewFakeClient(objs...)
			b := WithClients(cl, &fakeDiscoveryClient{}, cl)

			// test
			b.cleanDeployments(context.Background())

			// verify
			persisted := &appsv1.Deployment{}
			err := cl.Get(context.Background(), types.NamespacedName{
				Namespace: dep.Namespace,
				Name:      dep.Name,
			}, persisted)
			require.NoError(t, err)

			// should the sidecar have been deleted?
			if tt.deleted {
				assert.Equal(t, 1, len(persisted.Spec.Template.Spec.Containers))
				assert.NotContains(t, persisted.Labels, inject.Label)
			} else {
				assert.Equal(t, 2, len(persisted.Spec.Template.Spec.Containers))
				assert.Contains(t, persisted.Labels, inject.Label)
			}
		})
	}
}

type fakeClient struct {
	client.Client
	CreateFunc func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
}

func customFakeClient() *fakeClient {
	c := fake.NewFakeClient()
	return &fakeClient{Client: c, CreateFunc: c.Create}
}

func (f *fakeClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	return f.CreateFunc(ctx, obj)
}

type fakeDiscoveryClient struct {
	discovery.DiscoveryInterface
	ServerGroupsFunc func() (apiGroupList *metav1.APIGroupList, err error)
}

func (d *fakeDiscoveryClient) ServerGroups() (apiGroupList *metav1.APIGroupList, err error) {
	if d.ServerGroupsFunc == nil {
		return &metav1.APIGroupList{}, nil
	}
	return d.ServerGroupsFunc()
}

func (d *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (resources *metav1.APIResourceList, err error) {
	return &metav1.APIResourceList{}, nil
}

func (d *fakeDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{}, nil
}

func (d *fakeDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{}, nil
}

func (d *fakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{}, nil
}

func (d *fakeDiscoveryClient) ServerVersion() (*version.Info, error) {
	return &version.Info{}, nil
}

func (d *fakeDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return &openapi_v2.Document{}, nil
}

func (d *fakeDiscoveryClient) RESTClient() restclient.Interface {
	return &fakeRest.RESTClient{}
}
