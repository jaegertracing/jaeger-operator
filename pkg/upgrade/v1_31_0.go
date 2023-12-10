package upgrade

import (
	"context"
	"fmt"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func upgrade1_31_0(ctx context.Context, c client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	// Delete ES instance if self-provisioned ES is used.
	// The newly created instance will use cert-management from EO operator.
	if v1.ShouldInjectOpenShiftElasticsearchConfiguration(jaeger.Spec.Storage) {
		es := esv1.Elasticsearch{
			ObjectMeta: metav1.ObjectMeta{
				// The only possible name
				Name:      "elasticsearch",
				Namespace: jaeger.Namespace,
			},
		}
		err := client.IgnoreNotFound(c.Delete(ctx, &es))
		if err != nil {
			return jaeger, fmt.Errorf("failed to delete Elasticsearch, deletion is needed for certificate migration to Elasticsearch operator: %w", err)
		}
	}

	return jaeger, nil
}
