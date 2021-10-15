package upgrade

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_27_0(_ context.Context, _ client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	fmt.Println("upgrading jaeger 1.27")
	// Set the SAR to empty to ignore normalization to a default SAR
	// This workaround prevents a breaking change in already deployed Jaeger instances
	if jaeger.Spec.Ingress.Openshift.SAR == nil {
		empty := " "
		jaeger.Spec.Ingress.Openshift.SAR = &empty
	}
	return jaeger, nil
}
