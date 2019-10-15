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
	b := WithClients(cl, dcl)

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
	b := WithClients(cl, dcl)

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

func TestAutoDetectFallback(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	// sanity check
	assert.False(t, viper.IsSet("platform"))
	assert.False(t, viper.IsSet("es-provision"))

	// set the error
	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return nil, fmt.Errorf("faked error")
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformKubernetes, viper.GetString("platform"))
	assert.False(t, viper.GetBool("es-provision"))
}

func TestAutoDetectOpenShift(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformAutoDetect)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

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
}

func TestAutoDetectKubernetes(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformAutoDetect)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

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
	b := WithClients(cl, dcl)

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
	b := WithClients(cl, dcl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.False(t, viper.GetBool("es-provision"))
}

func TestAutoDetectEsProvisionWithEsOperator(t *testing.T) {
	// prepare
	viper.Set("es-provision", v1.FlagProvisionElasticsearchAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

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
	assert.True(t, viper.GetBool("es-provision"))
}

func TestAutoDetectKafkaProvisionNoKafkaOperator(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("kafka-provision", v1.FlagProvisionKafkaAuto)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.False(t, viper.GetBool("kafka-provision"))
}

func TestAutoDetectKafkaProvisionWithKafkaOperator(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

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
	assert.True(t, viper.GetBool("kafka-provision"))
}

func TestAutoDetectKafkaExplicitTrue(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaTrue)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.True(t, viper.GetBool("kafka-provision"))
}

func TestAutoDetectKafkaExplicitFalse(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaFalse)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.False(t, viper.GetBool("kafka-provision"))
}

func TestAutoDetectKafkaDefaultNoOperator(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.False(t, viper.GetBool("kafka-provision"))
}

func TestAutoDetectKafkaDefaultWithOperator(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

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
	assert.True(t, viper.GetBool("kafka-provision"))
}

func TestNoAuthDelegatorAvailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b := WithClients(cl, dcl)

	// test
	b.detectClusterRoles()

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
	b := WithClients(cl, dcl)

	// test
	b.detectClusterRoles()
	assert.False(t, viper.GetBool("auth-delegator-available"))

	cl.CreateFunc = cl.Client.Create
	b.detectClusterRoles()
	assert.True(t, viper.GetBool("auth-delegator-available"))
}

func TestAuthDelegatorBecomesUnavailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	b := WithClients(cl, dcl)

	// test
	b.detectClusterRoles()
	assert.True(t, viper.GetBool("auth-delegator-available"))

	cl.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b.detectClusterRoles()
	assert.False(t, viper.GetBool("auth-delegator-available"))
}

func TestCleanDeployments(t *testing.T) {
	cl := customFakeClient()
	cl.CreateFunc = cl.Client.Create
	dcl := &fakeDiscoveryClient{}

	jaeger1 := v1.NewJaeger(types.NamespacedName{
		Name:      "TestDeletedInstance",
		Namespace: "TestNS",
	})

	jaeger2 := v1.NewJaeger(types.NamespacedName{
		Name:      "TestDeletedInstance2",
		Namespace: "TestNS",
	})

	dep1 := &appsv1.Deployment{
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
	dep2 := &appsv1.Deployment{
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

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	dep1.Name = "mydep1"
	dep1.Annotations = map[string]string{inject.Annotation: jaeger1.Name}
	dep1 = inject.Sidecar(jaeger1, dep1)

	dep2.Name = "mydep2"
	dep2.Annotations = map[string]string{inject.Annotation: jaeger2.Name}
	dep2 = inject.Sidecar(jaeger2, dep2)

	require.Equal(t, len(dep1.Spec.Template.Spec.Containers), 2)
	require.Equal(t, len(dep2.Spec.Template.Spec.Containers), 2)

	err := cl.Create(context.TODO(), dep1)
	require.NoError(t, err)
	err = cl.Create(context.TODO(), dep2)
	require.NoError(t, err)
	err = cl.Create(context.TODO(), jaeger2)
	require.NoError(t, err)

	b := WithClients(cl, dcl)
	b.cleanDeployments()
	persisted1 := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Namespace: dep1.Namespace,
		Name:      dep1.Name,
	}, persisted1)
	require.NoError(t, err)
	assert.Equal(t, len(persisted1.Spec.Template.Spec.Containers), 1)
	assert.NotContains(t, persisted1.Labels, inject.Label)

	persisted2 := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Namespace: dep2.Namespace,
		Name:      dep2.Name,
	}, persisted2)
	require.NoError(t, err)
	assert.Equal(t, len(persisted2.Spec.Template.Spec.Containers), 2)
	assert.Contains(t, persisted2.Labels, inject.Label)
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
