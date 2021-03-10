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

package strategy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/internal/config"
)

func TestNewControllerForProduction(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Strategy = v2.DeploymentStrategyProduction
	cfg := config.New()
	ctrl := For(context.TODO(), logf.Log.WithName("unit-tests"), cfg, *jaeger)
	assert.Equal(t, ctrl.Type, v2.DeploymentStrategyProduction)
}

func TestNewControllerForProductionAsDefault(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	cfg := config.New()
	ctrl := For(context.TODO(), logf.Log.WithName("unit-tests"), cfg, *jaeger)
	assert.Equal(t, ctrl.Type, v2.DeploymentStrategyProduction)
}

func TestNewControllerForAllInOneAsExplicitValue(t *testing.T) {
	jaeger := v2.NewJaeger(types.NamespacedName{Name: "my-instance"})
	cfg := config.New()
	jaeger.Spec.Strategy = v2.DeploymentStrategyAllInOne
	ctrl := For(context.TODO(), logf.Log.WithName("unit-tests"), cfg, *jaeger)
	assert.Equal(t, ctrl.Type, v2.DeploymentStrategyAllInOne)
}
