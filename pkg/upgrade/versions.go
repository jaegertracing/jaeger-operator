package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type upgradeFunction = func(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error)

var (
	upgrades = map[string]upgradeFunction{
		"1.15.0": upgrade1_15_0,
		"1.17.0": upgrade1_17_0,
		"1.18.0": upgrade1_18_0,
		"1.20.0": upgrade1_20_0,
		// "1.22.0": upgrade1_22_0, // enable once 1.22 is to be released
	}
)
