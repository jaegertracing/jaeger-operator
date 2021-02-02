// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

func TestNewObjectsOnReconciliation(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	logger := logf.Log.WithName("unit-tests")

	reconciler := JaegerReconciler{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
	}

	created := v2.NewJaeger(nsn)
	created.Spec.Strategy = "production"

	err := k8sClient.Create(context.Background(), created)
	require.NoError(t, err)
	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}

	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
	}
	// verify that we have at least one object for each of the types we create
	// whether we have the right ones is up to the specific tests for each type
	{
		list := &otelv1alpha1.OpenTelemetryCollectorList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}

	require.NoError(t, k8sClient.Delete(context.Background(), created))

}

func TestRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)

	reconciler := controllers.NewReconciler(controllers.Params{})

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	assert.NoError(t, err)
}
