package v1

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	_ webhook.Defaulter = &Jaeger{}
	_ webhook.Validator = &Jaeger{}
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
		{
			name: "missing tls enable flag",
			j: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type:    JaegerMemoryStorage,
						Options: NewOptions(map[string]interface{}{"stuff.tls.test": "something"}),
					},
				},
			},
			expected: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Type: JaegerMemoryStorage,
						Options: NewOptions(
							map[string]interface{}{
								"stuff.tls.test":    "something",
								"stuff.tls.enabled": "true",
							},
						),
						Elasticsearch: ElasticsearchSpec{
							Name: defaultElasticsearchName,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, esv1.AddToScheme(scheme.Scheme))
			require.NoError(t, AddToScheme(scheme.Scheme))
			fakeCl := fake.NewClientBuilder().WithRuntimeObjects(test.objs...).Build()
			cl = fakeCl

			test.j.Default()
			assert.Equal(t, test.expected, test.j)
		})
	}
}

func TestValidateDelete(t *testing.T) {
	warnings, err := new(Jaeger).ValidateDelete()
	assert.Nil(t, warnings)
	require.NoError(t, err)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name         string
		objsToCreate []runtime.Object
		current      *Jaeger
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
		{
			name: "missing tls options",
			current: &Jaeger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "project1",
				},
				Spec: JaegerSpec{
					Storage: JaegerStorageSpec{
						Options: NewOptions(map[string]interface{}{
							"something.tls.else": "fails",
						}),
						Type: JaegerMemoryStorage,
					},
				},
			},
			err: `tls flags incomplete, got: [--something.tls.else=fails]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, esv1.AddToScheme(scheme.Scheme))
			require.NoError(t, AddToScheme(scheme.Scheme))
			fakeCl := fake.NewClientBuilder().WithRuntimeObjects(test.objsToCreate...).Build()
			cl = fakeCl

			warnings, err := test.current.ValidateCreate()
			if test.err != "" {
				require.Error(t, err)
				assert.Equal(t, test.err, err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Nil(t, warnings)
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

func TestGetAdditionalTLSFlags(t *testing.T) {
	tt := []struct {
		name   string
		args   []string
		expect map[string]interface{}
	}{
		{
			name:   "no tls flag",
			args:   []string{"--something.else"},
			expect: nil,
		},
		{
			name:   "already enabled",
			args:   []string{"--something.tls.enabled=true", "--something.tls.else=abc"},
			expect: nil,
		},
		{
			name:   "is disabled",
			args:   []string{"--tls.enabled=false", "--something.else", "--something.tls.else=abc"},
			expect: nil,
		},
		{
			name: "must be enabled",
			args: []string{"--something.tls.else=abc"},
			expect: map[string]interface{}{
				"something.tls.enabled": "true",
			},
		},
		{
			// NOTE: we want to avoid something like:
			// --kafka.consumer.authentication=tls.enabled=true
			name: "enable consumer tls",
			args: []string{
				"--es.server-urls=http://elasticsearch:9200",
				"--kafka.consumer.authentication=tls",
				"--kafka.consumer.brokers=my-cluster-kafka-bootstrap:9093",
				"--kafka.consumer.tls.ca=/var/run/secrets/cluster-ca/ca.crt",
				"--kafka.consumer.tls.cert=/var/run/secrets/kafkauser/user.crt",
				"--kafka.consumer.tls.key=/var/run/secrets/kafkauser/user.key",
			},
			expect: map[string]interface{}{
				"kafka.consumer.tls.enabled": "true",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := getAdditionalTLSFlags(tc.args)
			if !cmp.Equal(tc.expect, got) {
				t.Error("err:", cmp.Diff(tc.expect, got))
			}
		})
	}
}
