package autoclean

import (
	"context"
	"testing"

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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

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
								{
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
			require.Len(t, dep.Spec.Template.Spec.Containers, 2)

			// prepare the list of existing objects
			objs := []runtime.Object{dep}
			if len(tt.jaegerNamespace) > 0 {
				jaeger.Namespace = tt.jaegerNamespace // now, it exists only in this namespace
				objs = append(objs, jaeger)
			}

			// prepare the client
			s := scheme.Scheme
			s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
			s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
			cl := fake.NewClientBuilder().WithRuntimeObjects(objs...).Build()
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
				assert.Len(t, persisted.Spec.Template.Spec.Containers, 1)
				assert.NotContains(t, persisted.Labels, inject.Label)
			} else {
				assert.Len(t, persisted.Spec.Template.Spec.Containers, 2)
				assert.Contains(t, persisted.Labels, inject.Label)
			}
		})
	}
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
