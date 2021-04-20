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
	"fmt"
	"strings"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/version"
)

// Image returns the image associated with the supplied image if defined, otherwise
// uses the parameter name to retrieve the value. If the parameter value does not
// include a tag/digest, the Jaeger version will be appended.
func Image(image, defaultImage string, ver version.Version) string {
	if image == "" {
		param := defaultImage
		if strings.IndexByte(param, ':') == -1 {
			image = fmt.Sprintf("%s:%s", param, version.Jaeger())
		} else {
			image = param
		}
	}
	return image
}

func Collector(instance v2.Jaeger) string {
	return fmt.Sprintf("%s-collector", instance.Name)
}

func Agent(instance v2.Jaeger) string {
	return fmt.Sprintf("%s-agent", instance.Name)
}

// Service builds the service name based on the instance.
func CollectorService(instance v2.Jaeger) string {
	return fmt.Sprintf("%s-collector-collector", instance.Name)
}

func CollectorHeadlessService(instance v2.Jaeger) string {
	return fmt.Sprintf("%s-headless", CollectorService(instance))
}
