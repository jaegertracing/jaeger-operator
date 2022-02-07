package v1

import (
	"fmt"
	"testing"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	tests := []struct {
		name     string
		objs     []runtime.Object
		j        *Jaeger
		expected *Jaeger
	}{
		{
			name: "set missing ES name",
			j: &Jaeger{
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Elasticsearch: ElasticsearchSpec{
							Name: "",
						},
					},
				},
			},
			expected: &Jaeger{
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Elasticsearch: ElasticsearchSpec{
							Name: "elasticsearch",
						},
					},
				},
			},
		},
		{
			name: "set ES node count",
			objs: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project1",
					},
				},
				&esv1.Elasticsearch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-es",
						Namespace: "project1",
					},
					Spec: esv1.ElasticsearchSpec{
						Nodes: []esv1.ElasticsearchNode{
							{
								NodeCount: 3,
							},
						},
					},
				},
			},
			j: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							DoNotProvision: true,
						},
					},
				},
			},
			expected: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							NodeCount:      3,
							DoNotProvision: true,
						},
					},
				},
			},
		},
		{
			name: "do not set ES node count",
			j: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							DoNotProvision: false,
							NodeCount:      1,
						},
					},
				},
			},
			expected: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							NodeCount:      1,
							DoNotProvision: false,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			esv1.AddToScheme(scheme.Scheme)
			AddToScheme(scheme.Scheme)
			fakeCl := fake.NewClientBuilder().WithRuntimeObjects(test.objs...).Build()
			cl = fakeCl

			test.j.Default()
			assert.Equal(t, test.expected, test.j)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name         string
		objsToCreate []runtime.Object
		current      *Jaeger
		old          *Jaeger
		err          string
	}{
		{
			name: "ES instance exists",
			objsToCreate: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project1",
					},
				},
				&esv1.Elasticsearch{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-es",
						Namespace: "project1",
					},
					Spec: esv1.ElasticsearchSpec{
						Nodes: []esv1.ElasticsearchNode{
							{
								NodeCount: 3,
							},
						},
					},
				},
			},
			current: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							DoNotProvision: true,
						},
					},
				},
			},
		},
		{
			name: "ES instance does not exist",
			objsToCreate: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project1",
					},
				},
			},
			current: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: "elasticsearch",
						Elasticsearch: ElasticsearchSpec{
							Name:           "my-es",
							DoNotProvision: true,
						},
					},
				},
			},
			err: `elasticsearch instance not found: elasticsearchs.logging.openshift.io "my-es" not found`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			esv1.AddToScheme(scheme.Scheme)
			AddToScheme(scheme.Scheme)
			fakeCl := fake.NewClientBuilder().WithRuntimeObjects(test.objsToCreate...).Build()
			cl = fakeCl

			err := test.current.ValidateUpdate(test.old)
			if test.err != "" {
				assert.Equal(t, test.err, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestShouldDeployElasticsearch(t *testing.T) {
	tests := []struct {
		j        JaegerStorageSpec
		expected bool
	}{
		{j: JaegerStorageSpec{}},
		{j: JaegerStorageSpec{Type: JaegerCassandraStorage}},
		{j: JaegerStorageSpec{Type: JaegerESStorage, Options: NewOptions(map[string]interface{}{"es.server-urls": "foo"})}},
		{j: JaegerStorageSpec{Type: JaegerESStorage}, expected: true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			assert.Equal(t, test.expected, ShouldInjectOpenShiftElasticsearchConfiguration(test.j))
		})
	}
}
