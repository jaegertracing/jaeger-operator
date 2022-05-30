package upgrade

import (
	"context"
	"fmt"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
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
		err := c.Delete(ctx, &es)
		if err != nil {
			// ignore the error if the requested Elasticsearch resource is not available to delete
			if errors.IsNotFound(err) {
				jaeger.Logger().WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
					"current":   jaeger.Status.Version,
				}).Debug("Requested 'elasticsearch' instance is not available to delete")
				return jaeger, nil
			}
			return jaeger, fmt.Errorf("failed to delete Elasticsearch, deletion is needed for certificate migration to Elasticsearch operator: %v", err)
		}
	}

	return jaeger, nil
}
