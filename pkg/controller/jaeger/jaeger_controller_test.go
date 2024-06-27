package jaeger

import (
	"context"
	"fmt"
	"testing"

	osconsolev1 "github.com/openshift/api/console/v1"
	osv1 "github.com/openshift/api/route/v1"
	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/kafka/v1beta2"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

type modifiedClient struct {
	client.Client

	counter   int
	listErr   error
	getErr    error
	updateErr error
}

func (u *modifiedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	u.counter++
	if u.updateErr != nil {
		return u.updateErr
	}
	return u.Client.Update(ctx, obj, opts...)
}

func (u *modifiedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if u.listErr != nil {
		return u.listErr
	}
	return u.Client.List(ctx, list, opts...)
}

func (u *modifiedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if u.getErr != nil {
		return u.getErr
	}
	return u.Client.Get(ctx, key, obj, opts...)
}

func TestReconcileSyncOnJaegerChanges(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestNewJaegerInstance",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		jaeger.Spec.Strategy = "custom-strategy"
		return strategy.S{}
	}

	errList := fmt.Errorf("no no list")
	r.rClient = &modifiedClient{
		Client:  cl,
		listErr: errList,
	}

	// test
	_, err := r.Reconcile(req)
	assert.Equal(t, errList, err)
}

func TestSyncOnJaegerChanges(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{
		Namespace: "observability",
		Name:      "my-instance",
	})

	objs := []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "ns-with-annotation",
			Annotations: map[string]string{
				inject.Annotation: "true",
			},
		}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-without-annotation",
				Namespace: "ns-with-annotation",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-annotation",
				Namespace: "ns-with-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
			},
		},

		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "ns-without-annotation",
		}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-without-annotation",
				Namespace: "ns-without-annotation",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-annotation",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-with-another-jaegers-label",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
				Labels: map[string]string{
					inject.Label: "some-other-jaeger",
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-affected-jaeger-label",
				Namespace: "ns-without-annotation",
				Annotations: map[string]string{
					inject.Annotation: "true",
				},
				Labels: map[string]string{
					inject.Label: jaeger.Name,
				},
			},
		},
	}

	var (
		errList   = fmt.Errorf("no no listing")
		errGet    = fmt.Errorf("no no get")
		errUpdate = fmt.Errorf("no no update")
	)

	cl := &modifiedClient{
		Client:  fake.NewClientBuilder().WithObjects(objs...).Build(),
		listErr: errList,
		getErr:  errGet,
	}

	err := syncOnJaegerChanges(cl, cl, jaeger.Name)
	assert.Equal(t, errList, err)
	cl.listErr = nil

	_ = syncOnJaegerChanges(cl, cl, jaeger.Name)
	assert.Equal(t, 3, cl.counter)
	cl.counter = 0
	cl.getErr = nil

	err = syncOnJaegerChanges(cl, cl, jaeger.Name)
	assert.Equal(t, 4, cl.counter)
	require.NoError(t, err)
	cl.counter = 0

	cl.updateErr = errUpdate
	err = syncOnJaegerChanges(cl, cl, jaeger.Name)
	assert.Equal(t, 1, cl.counter)
	assert.Equal(t, errUpdate, err)
}

func TestNewJaegerInstance(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{
		Name: "TestNewJaegerInstance",
	}

	objs := []client.Object{
		v1.NewJaeger(nsn),
	}

	req := reconcile.Request{
		NamespacedName: nsn,
	}

	r, cl := getReconciler(objs)
	r.strategyChooser = func(ctx context.Context, jaeger *v1.Jaeger) strategy.S {
		jaeger.Spec.Strategy = "custom-strategy"
		return strategy.S{}
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1.Jaeger{}
	err = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.Equal(t, persisted.Name, nsn.Name)
	require.NoError(t, err)

	// these are filled with default values
	assert.Equal(t, v1.DeploymentStrategyAllInOne, persisted.Spec.Strategy)

	// the status object got updated as well
	assert.Equal(t, v1.JaegerPhaseRunning, persisted.Status.Phase)
}

func TestDeletedInstance(t *testing.T) {
	// prepare

	// we should just not fail, as there won't be anything to do
	// all our objects should have an OwnerReference, so that when the jaeger object is deleted, the owned objects are deleted as well
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestDeletedInstance"})
	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)

	// no known objects
	cl := fake.NewClientBuilder().Build()

	r := &ReconcileJaeger{client: cl, scheme: s, rClient: cl}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      jaeger.Name,
			Namespace: jaeger.Namespace,
		},
	}

	// test
	res, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	assert.False(t, res.Requeue, "We don't requeue for now")

	persisted := &v1.Jaeger{}
	_ = cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.NotEmpty(t, jaeger.Name)
	assert.Empty(t, persisted.Name) // this means that the object wasn't found
}

func TestSetOwnerOnNewInstance(t *testing.T) {
	// prepare
	viper.Set(v1.ConfigIdentity, "my-identity")
	defer viper.Reset()

	nsn := types.NamespacedName{Name: "my-instance"}
	jaeger := v1.NewJaeger(nsn)

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)
	cl := fake.NewClientBuilder().WithStatusSubresource(jaeger).WithObjects(jaeger).Build()

	r := &ReconcileJaeger{client: cl, scheme: s, rClient: cl}
	req := reconcile.Request{NamespacedName: nsn}

	// test
	_, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	persisted := &v1.Jaeger{}
	cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.NotNil(t, persisted.Labels)
	assert.Equal(t, "my-identity", persisted.Labels[v1.LabelOperatedBy])
}

func TestSkipOnNonOwnedCR(t *testing.T) {
	// prepare
	viper.Set(v1.ConfigIdentity, "my-identity")
	defer viper.Reset()

	nsn := types.NamespacedName{Name: "my-instance"}
	jaeger := v1.NewJaeger(nsn)
	jaeger.Labels = map[string]string{
		v1.LabelOperatedBy: "another-identity",
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)
	cl := fake.NewClientBuilder().WithObjects(jaeger).Build()
	r := &ReconcileJaeger{client: cl, scheme: s, rClient: cl}
	req := reconcile.Request{NamespacedName: nsn}

	// test
	_, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	persisted := &v1.Jaeger{}
	cl.Get(context.Background(), req.NamespacedName, persisted)
	assert.NotNil(t, persisted.Labels)

	// the only way to reliably test this is to verify that the operator didn't attempt to set the ownership field
	assert.Equal(t, "another-identity", persisted.Labels[v1.LabelOperatedBy])
	assert.Equal(t, v1.JaegerPhase(""), persisted.Status.Phase)
}

func TestGetResourceFromNonCachedClient(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}
	jaeger := v1.NewJaeger(nsn)

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, jaeger)

	// simulates the case where the cache is stale: the instance has been deleted (client) but the cache hasn't been updated (cachedClient)
	// we trigger the reconciliation and expect it to finish without errors, while we expect to not have an instance afterwards
	// if the code is using the cached client, we would end up either with an error (trying to update an instance that does not exist)
	// or we'd end up with an instance
	cachedClient := fake.NewClientBuilder().WithObjects(jaeger).Build()
	client := fake.NewClientBuilder().Build()

	r := &ReconcileJaeger{client: cachedClient, scheme: s, rClient: client}
	req := reconcile.Request{NamespacedName: nsn}

	// test
	_, err := r.Reconcile(req)

	// verify
	require.NoError(t, err)
	persisted := &v1.Jaeger{}
	err = client.Get(context.Background(), req.NamespacedName, persisted)
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func TestGetSecretsForNamespace(t *testing.T) {
	r := &ReconcileJaeger{}

	secretOne := createSecret("foo", "secretOne")
	secretTwo := createSecret("foo", "secretTwo")

	secrets := []corev1.Secret{secretOne, secretTwo}
	filteredSecrets := r.getSecretsForNamespace(secrets, "foo")
	assert.Len(t, filteredSecrets, 2)

	secretThree := createSecret("bar", "secretThree")
	secrets = append(secrets, secretThree)
	filteredSecrets = r.getSecretsForNamespace(secrets, "bar")
	assert.Len(t, filteredSecrets, 1)
	assert.Contains(t, filteredSecrets, secretThree)
}

func TestElasticsearchProvisioning(t *testing.T) {
	namespacedName := types.NamespacedName{Name: "prod", Namespace: "jaeger"}
	j := v1.NewJaeger(namespacedName)
	j.Spec.Storage.Type = "elasticsearch"
	j.Spec.Storage.Elasticsearch.Name = "elasticserach"
	j.Spec.Storage.Elasticsearch.NodeCount = 1

	reconciler, cl := getReconciler([]client.Object{j})

	req := reconcile.Request{NamespacedName: namespacedName}
	result, err := reconciler.Reconcile(req)
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	secrets := &corev1.SecretList{}
	err = cl.List(context.Background(), secrets, client.InNamespace("jaeger"))
	require.NoError(t, err)
	assert.Len(t, secrets.Items, 4)
	assert.NotNil(t, getSecret("prod-jaeger-elasticsearch", *secrets))
	assert.NotNil(t, getSecret("prod-master-certs", *secrets))
	assert.NotNil(t, getSecret("prod-curator", *secrets))
	assert.NotNil(t, getSecret("elasticsearch", *secrets))
}

func getSecret(name string, secrets corev1.SecretList) *corev1.Secret {
	for _, s := range secrets.Items {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

func createSecret(secretNamespace, secretName string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func getReconciler(objs []client.Object) (*ReconcileJaeger, client.Client) {
	s := scheme.Scheme

	// OpenShift Route
	osv1.Install(s)

	// OpenShift ConsoleLink
	osconsolev1.Install(s)

	// Jaeger
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})

	// Jaeger's Elasticsearch
	s.AddKnownTypes(v1.GroupVersion, &esv1.Elasticsearch{}, &esv1.ElasticsearchList{})

	// Kafka
	s.AddKnownTypes(v1beta2.GroupVersion, &v1beta2.Kafka{}, &v1beta2.KafkaList{}, &v1beta2.KafkaUser{}, &v1beta2.KafkaUserList{})

	cl := fake.NewClientBuilder().WithScheme(s).WithStatusSubresource(objs...).WithObjects(objs...).Build()

	r := New(cl, cl, s)
	r.certGenerationScript = "../../../scripts/cert_generation.sh"
	return r, cl
}
