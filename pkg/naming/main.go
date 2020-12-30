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

	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/internal/version"
)

// Image returns the image associated with the supplied image if defined, otherwise
// uses the parameter name to retrieve the value. If the parameter value does not
// include a tag/digest, the Jaeger version will be appended.
func Image(image, param string) string {
	if image == "" {
		param := viper.GetString(param)
		if strings.IndexByte(param, ':') == -1 {
			image = fmt.Sprintf("%s:%s", param, version.Get().Jaeger)
		} else {
			image = param
		}
	}
	return image
}
