package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

func TestKafkaUserCreate(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetKafkaIntegration(autodetect.KafkaOperatorIntegrationYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestKafkaUserCreate",
		Namespace: "tenant1",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		s := strategy.New().WithKafkaUsers([]kafkav1beta2.KafkaUser{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jaeger.Name,
				Namespace: jaeger.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaUserStatus{
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
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1beta2.KafkaUser{}
	persistedName := types.NamespacedName{
		Name:      nsn.Name,
		Namespace: nsn.Namespace,
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Equal(t, persistedName.Name, persisted.GetName())
	require.NoError(t, err)
}

func TestKafkaUserUpdate(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetKafkaIntegration(autodetect.KafkaOperatorIntegrationYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "TestKafkaUserUpdate",
		Namespace: "tenant1",
	}

	orig := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nsn.Name,
			Namespace:   nsn.Namespace,
			Annotations: map[string]string{"key": "value"},
			Labels: map[string]string{
				"app.kubernetes.io/instance":   nsn.Name,
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
		},
		Status: kafkav1beta2.KafkaUserStatus{
			Conditions: []kafkav1beta2.KafkaStatusCondition{{
				Type:   "Ready",
				Status: "True",
			}},
		},
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		kafkaUpdated := v1beta2.KafkaUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:        nsn.Name,
				Namespace:   nsn.Namespace,
				Annotations: map[string]string{"key": "new-value"},
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaUserStatus{
				Conditions: []kafkav1beta2.KafkaStatusCondition{{
					Type:   "Ready",
					Status: "True",
				}},
			},
		}

		s := strategy.New().WithKafkaUsers([]v1beta2.KafkaUser{kafkaUpdated})
		return s
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &v1beta2.KafkaUser{}
	persistedName := types.NamespacedName{
		Name:      orig.GetName(),
		Namespace: orig.GetNamespace(),
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, "new-value", persisted.Annotations["key"])
}

func TestKafkaUserDelete(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetKafkaIntegration(autodetect.KafkaOperatorIntegrationYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name: "TestKafkaUserDelete",
	}

	orig := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   nsn.Name,
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
		},
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		&orig,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		return strategy.S{}
	}

	// test
	_, err := r.Reconcile(reconcile.Request{NamespacedName: nsn})
	require.NoError(t, err)

	// verify
	persisted := &v1beta2.KafkaUser{}
	persistedName := types.NamespacedName{
		Name:      orig.GetName(),
		Namespace: orig.GetNamespace(),
	}
	err = cl.Get(context.Background(), persistedName, persisted)
	assert.Empty(t, persisted.GetName())
	require.Error(t, err) // not found
}

func TestKafkaUserCreateExistingNameInAnotherNamespace(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetKafkaIntegration(autodetect.KafkaOperatorIntegrationYes)
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "tenant1",
	}
	nsnExisting := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "tenant2",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
		v1.NewJaeger(nsnExisting),
		&v1beta2.KafkaUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsnExisting.Name,
				Namespace: nsnExisting.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsnExisting.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaUserStatus{
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
		s := strategy.New().WithKafkaUsers([]v1beta2.KafkaUser{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/instance":   nsn.Name,
					"app.kubernetes.io/managed-by": "jaeger-operator",
				},
			},
			Status: kafkav1beta2.KafkaUserStatus{
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
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1beta2.KafkaUser{}
	err = cl.Get(context.Background(), nsn, persisted)
	require.NoError(t, err)
	assert.Equal(t, nsn.Name, persisted.GetName())
	assert.Equal(t, nsn.Namespace, persisted.GetNamespace())

	persistedExisting := &v1beta2.KafkaUser{}
	err = cl.Get(context.Background(), nsnExisting, persistedExisting)
	require.NoError(t, err)
	assert.Equal(t, nsnExisting.Name, persistedExisting.GetName())
	assert.Equal(t, nsnExisting.Namespace, persistedExisting.GetNamespace())
}
