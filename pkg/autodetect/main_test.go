package autodetect

import (
	"context"
	"fmt"
	"testing"
	"time"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	fakeRest "k8s.io/client-go/rest/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestStart(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	// sanity check
	assert.False(t, viper.IsSet("auth-delegator-available"))

	// prepare
	dcl := &fakeDiscoveryClient{}
	cl := fake.NewClientBuilder().Build()
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
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	// prepare
	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
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

func TestAutoDetectWithServerGroupsError(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)

	// sanity check
	assert.False(t, viper.IsSet("platform"))
	assert.False(t, viper.IsSet("es-provision"))

	// set the error

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{}, fmt.Errorf("faked error")
	}

	// Check initial value of "platform"
	assert.Equal(t, "", viper.GetString("platform"))

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, "", viper.GetString("platform"))
	assert.False(t, viper.GetBool("es-provision"))
}

func TestAutoDetectWithServerResourcesForGroupVersionError(t *testing.T) {
	// prepare
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)

	// sanity check
	assert.False(t, viper.IsSet("platform"))
	assert.False(t, viper.IsSet("es-provision"))

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name: "route.openshift.io",
		}}}, nil
	}

	// set the error
	dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
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
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)

	dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
		return &metav1.APIResourceList{GroupVersion: "route.openshift.io/v1"}, nil
	}

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name: "route.openshift.io",
		}}}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagPlatformOpenShift, viper.GetString("platform"))

	// set the error
	dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name: "logging.openshift.io",
		}}}, nil
	}

	t.Run("kind Elasticsearch", func(t *testing.T) {
		dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
			return &metav1.APIResourceList{
				GroupVersion: "logging.openshift.io/v1",
				APIResources: []metav1.APIResource{
					{
						Kind: "Elasticsearch",
					},
				},
			}, nil
		}
		b.autoDetectCapabilities()
		assert.Equal(t, v1.FlagProvisionElasticsearchYes, viper.GetString("es-provision"))
	})

	t.Run("no kind Elasticsearch", func(t *testing.T) {
		dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
			return &metav1.APIResourceList{
				GroupVersion: "logging.openshift.io/v1",
				APIResources: []metav1.APIResource{
					{
						Kind: "Kibana",
					},
				},
			}, nil
		}
		b.autoDetectCapabilities()
		assert.Equal(t, v1.FlagProvisionElasticsearchNo, viper.GetString("es-provision"))
	})
}

func TestAutoDetectKafkaProvisionNoKafkaOperator(t *testing.T) {
	// prepare
	viper.Set("kafka-provision", v1.FlagProvisionKafkaAuto)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)

	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name: "kafka.strimzi.io",
		}}}, nil
	}

	dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
		return &metav1.APIResourceList{GroupVersion: "kafka.strimzi.io/v1"}, nil
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
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
	cl := fake.NewClientBuilder().Build()
	b := WithClients(cl, dcl, cl)
	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name: "kafka.strimzi.io",
		}}}, nil
	}

	dcl.ServerResourcesForGroupVersionFunc = func(_ string) (apiGroupList *metav1.APIResourceList, err error) {
		return &metav1.APIResourceList{GroupVersion: "kafka.strimzi.io/v1"}, nil
	}

	// test
	b.autoDetectCapabilities()

	// verify
	assert.Equal(t, v1.FlagProvisionKafkaYes, viper.GetString("kafka-provision"))
}

func TestAutoDetectCronJobsVersion(t *testing.T) {
	apiGroupVersions := []string{v1.FlagCronJobsVersionBatchV1, v1.FlagCronJobsVersionBatchV1Beta1}
	for _, apiGroup := range apiGroupVersions {
		dcl := &fakeDiscoveryClient{}
		cl := fake.NewFakeClient() // nolint:staticcheck
		b := WithClients(cl, dcl, cl)
		dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
			return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
				Name:     apiGroup,
				Versions: []metav1.GroupVersionForDiscovery{{Version: apiGroup}},
			}}}, nil
		}

		dcl.ServerResourcesForGroupVersionFunc = func(requestedApiVersion string) (apiGroupList *metav1.APIResourceList, err error) {
			if requestedApiVersion == apiGroup {
				apiResourceList := &metav1.APIResourceList{GroupVersion: apiGroup, APIResources: []metav1.APIResource{{Name: "cronjobs"}}}
				return apiResourceList, nil
			}
			return &metav1.APIResourceList{}, nil
		}

		// test
		b.autoDetectCapabilities()

		// verify
		assert.Equal(t, apiGroup, viper.GetString(v1.FlagCronJobsVersion))
	}
}

func TestAutoDetectAutoscalingVersion(t *testing.T) {
	apiGroupVersions := []string{v1.FlagAutoscalingVersionV2, v1.FlagAutoscalingVersionV2Beta2}
	for _, apiGroup := range apiGroupVersions {
		dcl := &fakeDiscoveryClient{}
		cl := fake.NewFakeClient() // nolint:staticcheck
		b := WithClients(cl, dcl, cl)
		dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
			return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
				Name:     apiGroup,
				Versions: []metav1.GroupVersionForDiscovery{{Version: apiGroup}},
			}}}, nil
		}

		dcl.ServerResourcesForGroupVersionFunc = func(requestedApiVersion string) (apiGroupList *metav1.APIResourceList, err error) {
			if requestedApiVersion == apiGroup {
				apiResourceList := &metav1.APIResourceList{GroupVersion: apiGroup, APIResources: []metav1.APIResource{{Name: "horizontalpodautoscalers"}}}
				return apiResourceList, nil
			}
			return &metav1.APIResourceList{}, nil
		}

		// test
		b.autoDetectCapabilities()

		// verify
		assert.Equal(t, apiGroup, viper.GetString(v1.FlagAutoscalingVersion))
		fmt.Printf("Test finished on [%s]\n", apiGroup)
	}

	// Check what happens when there ServerResourcesForGroupVersion returns error
	dcl := &fakeDiscoveryClient{}
	cl := fake.NewFakeClient() // nolint:staticcheck
	b := WithClients(cl, dcl, cl)
	dcl.ServerGroupsFunc = func() (apiGroupList *metav1.APIGroupList, err error) {
		return &metav1.APIGroupList{Groups: []metav1.APIGroup{{
			Name:     v1.FlagAutoscalingVersionV2,
			Versions: []metav1.GroupVersionForDiscovery{{Version: v1.FlagAutoscalingVersionV2}},
		}}}, nil
	}

	dcl.ServerResourcesForGroupVersionFunc = func(requestedApiVersion string) (apiGroupList *metav1.APIResourceList, err error) {
		return &metav1.APIResourceList{}, fmt.Errorf("Fake error")
	}

	// test
	b.autoDetectCapabilities()
}

func TestSkipAuthDelegatorNonOpenShift(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformKubernetes)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.detectClusterRoles(context.Background())

	// verify
	assert.False(t, viper.IsSet("auth-delegator-available"))
}

func TestNoAuthDelegatorAvailable(t *testing.T) {
	// prepare
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
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
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	cl.CreateFunc = func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
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
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	dcl := &fakeDiscoveryClient{}
	cl := customFakeClient()
	b := WithClients(cl, dcl, cl)

	// test
	b.detectClusterRoles(context.Background())
	assert.True(t, viper.GetBool("auth-delegator-available"))

	cl.CreateFunc = func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
		return fmt.Errorf("faked error")
	}
	b.detectClusterRoles(context.Background())
	assert.False(t, viper.GetBool("auth-delegator-available"))
}

type fakeClient struct {
	client.Client
	CreateFunc func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
}

func customFakeClient() *fakeClient {
	c := fake.NewClientBuilder().Build()
	return &fakeClient{Client: c, CreateFunc: c.Create}
}

func (f *fakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return f.CreateFunc(ctx, obj)
}

type fakeDiscoveryClient struct {
	discovery.DiscoveryInterface
	ServerGroupsFunc                   func() (apiGroupList *metav1.APIGroupList, err error)
	ServerResourcesForGroupVersionFunc func(groupVersion string) (resources *metav1.APIResourceList, err error)
}

func (d *fakeDiscoveryClient) ServerGroups() (apiGroupList *metav1.APIGroupList, err error) {
	if d.ServerGroupsFunc == nil {
		return &metav1.APIGroupList{}, nil
	}
	return d.ServerGroupsFunc()
}

func (d *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (resources *metav1.APIResourceList, err error) {
	if d.ServerGroupsFunc == nil {
		return &metav1.APIResourceList{}, nil
	}
	return d.ServerResourcesForGroupVersionFunc(groupVersion)
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
