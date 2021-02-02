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

package normalize

import (
	"context"

	"go.opentelemetry.io/otel"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/instrument"
)

// normalize changes the incoming Jaeger object so that the defaults are applied when
// needed and incompatible options are cleaned.
func Jaeger(ctx context.Context, jaeger v2.Jaeger) v2.Jaeger {
	tracer := otel.GetTracerProvider().Tracer(instrument.ReconciliationTracer)
	_, span := tracer.Start(ctx, "normalize")
	defer span.End()

	// we need a name!
	if jaeger.Name == "" {
		//		jaeger.Logger().Info("This Jaeger instance was created without a name. Applying a default name.")
		jaeger.Name = "my-jaeger"
	}

	// normalize the deployment strategy
	if jaeger.Spec.Strategy != v2.DeploymentStrategyProduction && jaeger.Spec.Strategy != v2.DeploymentStrategyStreaming {
		jaeger.Spec.Strategy = v2.DeploymentStrategyAllInOne
	}

	return jaeger
}
