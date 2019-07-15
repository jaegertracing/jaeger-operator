package autodetect

import (
	"context"
	"fmt"
	"testing"
	"time"

	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	fakeRest "k8s.io/client-go/rest/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
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
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("platform", v1.FlagPlatformAutoDetect)

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
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("platform", v1.FlagPlatformAutoDetect)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformKubernetes, viper.GetString("platform"))
}

func TestExplicitPlatform(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("platform", v1.FlagPlatformOpenShift)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformOpenShift, viper.GetString("platform"))
}

func TestAutoDetectEsProvisionNoEsOperator(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("es-provision", v1.FlagProvisionElasticsearchAuto)

	// test
	b.autoDetectCapabilities()

	// verify
	assert.False(t, viper.GetBool("es-provision"))
}

func TestAutoDetectEsProvisionWithEsOperator(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient()
	b := WithClients(cl, dcl)

	viper.Set("es-provision", v1.FlagProvisionElasticsearchAuto)

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

func TestNoAuthDelegatorAvailable(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
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
	cl.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
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

	cl.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
		return fmt.Errorf("faked error")
	}
	b.detectClusterRoles()
	assert.False(t, viper.GetBool("auth-delegator-available"))
}

type fakeClient struct {
	client.Client
	CreateFunc func(ctx context.Context, obj runtime.Object) error
}

func customFakeClient() *fakeClient {
	c := fake.NewFakeClient()
	return &fakeClient{Client: c, CreateFunc: c.Create}
}

func (f *fakeClient) Create(ctx context.Context, obj runtime.Object) error {
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
