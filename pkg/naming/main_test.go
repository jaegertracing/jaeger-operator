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

package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/version"
)

func TestImageNameSupplied(t *testing.T) {
	assert.Equal(t, "org/actual-image:1.2.3", Image("org/actual-image:1.2.3", "org/default-image", version.Get()))
}

func TestImageNameParamNoTag(t *testing.T) {
	assert.Equal(t, "org/default-image:"+version.Get().Jaeger, Image("", "org/default-image", version.Get()))
}

func TestImageNameParamWithTag(t *testing.T) {
	assert.Equal(t, "org/default-image:1.2.3", Image("", "org/default-image:1.2.3", version.Get()))
}

func TestImageNameParamWithDigest(t *testing.T) {
	defaultImage := "org/custom-image@sha256:2a7ef4373262fa5fa3b3eaac86015650f8f3eee65d6e2674df931657873e318e"
	assert.Equal(t, defaultImage, Image("", defaultImage, version.Get()))
}

func TestImageNameParamDefaultNoTag(t *testing.T) {
	assert.Equal(t, "org/default-image:"+version.Get().Jaeger, Image("", "org/default-image", version.Get()))
}

func TestImageNameParamDefaultWithTag(t *testing.T) {
	assert.Equal(t, "org/default-image:1.2.3", Image("", "org/default-image:1.2.3", version.Get()))
}

func TestCollectorName(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Equal(t, "my-instance-collector", Collector(*jaeger))
}

func TestAgentName(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Equal(t, "my-instance-agent", Agent(*jaeger))
}

func TestCollectorService(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Equal(t, "my-instance-collector-collector", CollectorService(*jaeger))
}

func TestCollectorHeadlessService(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Equal(t, "my-instance-collector-collector-headless", CollectorHeadlessService(*jaeger))
}
