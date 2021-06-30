package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestKafkaCreate(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestKafkaCreate",
		Namespace: "tenant1",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithKafkas([]v1beta2.Kafka{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jaeger.Name,
				Namespace: jaeger.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaStatus{
				Conditions: []kafkav1beta2.KafkaStatusCondition{{
					Type:   "Ready",
					Status: "True",
				}},
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1beta2.Kafka{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.GetName())
	assert.NoError(t, err)
}

func TestKafkaUpdate(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestKafkaUpdate",
		Namespace: "tenant1",
	}

	orig := v1beta2.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nsn.Name,
			Namespace:   nsn.Namespace,
			Annotations: map[string]string{"key": "value"},
			Labels: map[string]string{
				"app.kubernetes.io/instance":   nsn.Name,
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
		},
		Status: kafkav1beta2.KafkaStatus{
			Conditions: []kafkav1beta2.KafkaStatusCondition{{
				Type:   "Ready",
				Status: "True",
			}},
		},
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		kafkaUpdated := v1beta2.Kafka{
			ObjectMeta: metav1.ObjectMeta{
				Name:        nsn.Name,
				Namespace:   nsn.Namespace,
				Annotations: map[string]string{"key": "new-value"},
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaStatus{
				Conditions: []kafkav1beta2.KafkaStatusCondition{{
					Type:   "Ready",
					Status: "True",
				}},
			},
		}

		s := strategy.New().WithKafkas([]v1beta2.Kafka{kafkaUpdated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &v1beta2.Kafka{}
	persistedName := types.NamespacedName{
		Name:      orig.GetName(),
		Namespace: orig.GetNamespace(),
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
}

func TestKafkaDelete(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestKafkaDelete",
		Namespace: "tenant1",
	}

	orig := v1beta2.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   nsn.Name,
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
		},
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	assert.NoError(t, err)

	// verify
	persisted := &v1beta2.Kafka{}
	persistedName := types.NamespacedName{
		Name:      orig.GetName(),
		Namespace: orig.GetNamespace(),
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.GetName())
	assert.Error(t, err) // not found
}

func TestKafkaCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	viper.SetDefault("kafka-provision", v1.FlagProvisionKafkaYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "tenant2",
	}

	objs := []runtime.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&v1beta2.Kafka{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsnExisting.Name,
				Namespace: nsnExisting.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsnExisting.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaStatus{
				Conditions: []kafkav1beta2.KafkaStatusCondition{{
					Type:   "Ready",
					Status: "True",
				}},
			},
		},
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithKafkas([]v1beta2.Kafka{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaStatus{
				Conditions: []kafkav1beta2.KafkaStatusCondition{{
					Type:   "Ready",
					Status: "True",
				}},
			},
		}})
		return s
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1beta2.Kafka{}
	err = cl.Get(context.Background(), nsn, persisted)
	assert.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.GetName())
	assert.Equal(t, nsn.Namespace, persisted.GetNamespace())

	persistedExisting := &v1beta2.Kafka{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	assert.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.GetName())
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.GetNamespace())
}
