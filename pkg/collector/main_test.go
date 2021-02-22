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

package collector

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/internal/config"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

func TestDefaultCollectorImage(t *testing.T) {
	cfg := config.New()
	jaeger := jaegertracingv2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	collector := Get(*jaeger, cfg)
	assert.Empty(t, jaeger.Spec.Collector.Image)
	assert.Equal(t, "otel/opentelemetry-collector:0.19.0", collector.Spec.Image)
}

func TestDefaultCollectorConfig(t *testing.T) {
	cfg := config.New()
	jaeger := jaegertracingv2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	otelCollector := Get(*jaeger, cfg)
	assert.Empty(t, jaeger.Spec.Collector.Config)
	defaultCfgString, _ := defaultConfig().String()
	assert.Equal(t, defaultCfgString, otelCollector.Spec.Config)
}
