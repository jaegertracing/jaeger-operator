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

	"github.com/jaegertracing/jaeger-operator/internal/version"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

func TestDefaultCollectorImage(t *testing.T) {
	viper.Set("jaeger-collector-image", "org/custom-collector-image")
	defer viper.Reset()

	jaeger := jaegertracingv2.NewJaeger(types.NamespacedName{Name: "my-instance"})

	collector := Get(*jaeger)
	assert.Empty(t, jaeger.Spec.Collector.Image)
	assert.Equal(t, "org/custom-collector-image:"+version.Get().Jaeger, collector.Spec.Image)
}

func TestDefaultCollectorConfig(t *testing.T) {
	jaeger := jaegertracingv2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	otelCollector := Get(*jaeger)
	assert.Empty(t, jaeger.Spec.Collector.Config)
	assert.Equal(t, DefaultConfig(), otelCollector.Spec.Config)
}

func TestCustomCollectorConfig(t *testing.T) {
	customConfig := "OTHER_VALUE"
	jaeger := jaegertracingv2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Collector.Config = customConfig
	otelCollector := Get(*jaeger)
	assert.Equal(t, customConfig, jaeger.Spec.Collector.Config)
	assert.Equal(t, customConfig, otelCollector.Spec.Config)
}
