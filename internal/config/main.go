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
	"github.com/spf13/pflag"

	"github.com/jaegertracing/jaeger-operator/internal/version"
)

// Config holds the static configuration for this operator.
type Config struct {
	// config state
	collectorImage string
	jaegerVersion  version.Version
}

// FlagSet binds the flags to the user-modifiable values of the operator's configuration.
func (c *Config) FlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("jaeger-operator", pflag.ExitOnError)
	fs.StringVar(&c.collectorImage,
		"jaeger-collector-image",
		c.collectorImage,
		"The default image to use for the Jaeger Collector when not specified in the individual custom resource (CR)",
	)

	return fs
}

func New(opts ...Option) Config {

	// TODO: Replace default image with jaeger image.
	// initialize with the default values
	o := options{
		version: version.Get(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	if len(o.collectorImage) == 0 {
		o.collectorImage = "otel/opentelemetry-collector:0.19.0"
	}

	return Config{
		collectorImage: o.collectorImage,
		jaegerVersion:  o.version,
	}
}

func (c *Config) CollectorImage() string {
	return c.collectorImage
}

func (c *Config) JaegerVersion() version.Version {
	return c.jaegerVersion
}
