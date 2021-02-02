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

package config

import (
	"testing"

	"github.com/jaegertracing/jaeger-operator/internal/version"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	// prepare
	cfg := New()
	// test
	assert.Equal(t, "otel/opentelemetry-collector:0.19.0", cfg.CollectorImage())
}

func TestConfigWithCustomCollectorImage(t *testing.T) {
	// prepare
	cfg := New()
	args := []string{
		"--jaeger-collector-image=otel/test-custom:1.1.1",
	}
	fs := pflag.NewFlagSet("main", pflag.ContinueOnError)
	fs.AddFlagSet(cfg.FlagSet())
	err := fs.Parse(args)
	assert.NoError(t, err)
	// test
	assert.Equal(t, "otel/test-custom:1.1.1", cfg.CollectorImage())
}

func TestOverrideVersion(t *testing.T) {
	// prepare
	v := version.Version{
		Jaeger: "the-version",
	}
	cfg := New(WithVersion(v))

	// test
	assert.Contains(t, cfg.JaegerVersion().Jaeger, "the-version")
}

func TestOverrideCollectorImage(t *testing.T) {
	// prepare
	cfg := New(WithCollectorImage("new-image"))

	// test
	assert.Contains(t, cfg.CollectorImage(), "new-image")
}
