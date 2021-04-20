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

package agent

import (
	"fmt"
	"testing"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/config"
	"github.com/jaegertracing/jaeger-operator/pkg/naming"
)

func TestDefaultAgent(t *testing.T) {
	// Prepare configuration
	cfg := config.New()
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	otelcollectorConfig := defaultConfig()
	jaegerExporter := otelcollectorConfig.GetJaegerExporter()
	require.NotNil(t, jaegerExporter)
	jaegerExporter.Endpoint = fmt.Sprintf("%s.%s.svc:14250", naming.CollectorHeadlessService(*jaeger), jaeger.Namespace)
	strCfg, err := otelcollectorConfig.String()
	require.NoError(t, err)

	otelCollector := Get(*jaeger, logf.Log.WithName("unit-tests"), cfg)

	// Test conditions
	assert.NotNil(t, otelCollector)
	assert.Equal(t, strCfg, otelCollector.Spec.Config)
	assert.Equal(t, "my-instance-agent", otelCollector.Name)
	assert.Equal(t, otelv1alpha1.ModeSidecar, otelCollector.Spec.Mode)
}

func TestDaemonSetAgent(t *testing.T) {
	// Prepare configuration
	cfg := config.New()
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	otelcollectorConfig := defaultConfig()
	jaegerExporter := otelcollectorConfig.GetJaegerExporter()
	require.NotNil(t, jaegerExporter)
	jaegerExporter.Endpoint = fmt.Sprintf("%s.%s.svc:14250", naming.CollectorHeadlessService(*jaeger), jaeger.Namespace)
	strCfg, err := otelcollectorConfig.String()
	require.NoError(t, err)

	jaeger.Spec.Agent.Strategy = v2.AgentDaemonSet
	otelCollector := Get(*jaeger, logf.Log.WithName("unit-tests"), cfg)

	assert.Empty(t, jaeger.Spec.Collector.Config)
	assert.NotNil(t, otelCollector)
	assert.Equal(t, strCfg, otelCollector.Spec.Config)
	assert.Equal(t, "my-instance-agent", otelCollector.Name)
	assert.Equal(t, otelv1alpha1.ModeDaemonSet, otelCollector.Spec.Mode)
}

func TestInvalidAgentMode(t *testing.T) {
	// Prepare configuration
	cfg := config.New()
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	otelcollectorConfig := defaultConfig()
	jaegerExporter := otelcollectorConfig.GetJaegerExporter()
	require.NotNil(t, jaegerExporter)
	jaegerExporter.Endpoint = fmt.Sprintf("%s.%s.svc:14250", naming.CollectorHeadlessService(*jaeger), jaeger.Namespace)
	strCfg, err := otelcollectorConfig.String()
	require.NoError(t, err)

	jaeger.Spec.Agent.Strategy = "invalid"
	otelCollector := Get(*jaeger, logf.Log.WithName("unit-tests"), cfg)

	assert.Empty(t, jaeger.Spec.Collector.Config)
	assert.NotNil(t, otelCollector)
	assert.Equal(t, strCfg, otelCollector.Spec.Config)
	assert.Equal(t, "my-instance-agent", otelCollector.Name)
	assert.Equal(t, otelv1alpha1.ModeSidecar, otelCollector.Spec.Mode)
}

func TestCustomAgentConfig(t *testing.T) {
	customConfig := "OTHER_VALUE"
	cfg := config.New()
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Agent.Config = customConfig
	otelCollector := Get(*jaeger, logf.Log.WithName("unit-tests"), cfg)
	assert.Equal(t, customConfig, jaeger.Spec.Agent.Config)
	assert.Equal(t, customConfig, otelCollector.Spec.Config)
}
