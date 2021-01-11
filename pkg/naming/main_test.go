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

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/internal/version"
)

func TestImageNameSupplied(t *testing.T) {
	viper.Set("test-image", "org/custom-image")
	defer viper.Reset()

	assert.Equal(t, "org/actual-image:1.2.3", Image("org/actual-image:1.2.3", "test-image"))
}

func TestImageNameParamNoTag(t *testing.T) {
	viper.Set("test-image", "org/custom-image")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image:"+version.Get().Jaeger, Image("", "test-image"))
}

func TestImageNameParamWithTag(t *testing.T) {
	viper.Set("test-image", "org/custom-image:1.2.3")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image:1.2.3", Image("", "test-image"))
}

func TestImageNameParamWithDigest(t *testing.T) {
	viper.Set("test-image", "org/custom-image@sha256:2a7ef4373262fa5fa3b3eaac86015650f8f3eee65d6e2674df931657873e318e")
	defer viper.Reset()

	assert.Equal(t, "org/custom-image@sha256:2a7ef4373262fa5fa3b3eaac86015650f8f3eee65d6e2674df931657873e318e", Image("", "test-image"))
}

func TestImageNameParamDefaultNoTag(t *testing.T) {
	viper.SetDefault("test-image", "org/default-image")
	defer viper.Reset()

	assert.Equal(t, "org/default-image:"+version.Get().Jaeger, Image("", "test-image"))
}

func TestImageNameParamDefaultWithTag(t *testing.T) {
	viper.SetDefault("test-image", "org/default-image:1.2.3")
	defer viper.Reset()

	assert.Equal(t, "org/default-image:1.2.3", Image("", "test-image"))
}
