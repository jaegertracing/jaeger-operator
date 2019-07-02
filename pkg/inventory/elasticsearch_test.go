package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

func TestElasticsearchInventory(t *testing.T) {
	toCreate := esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: esv1.ElasticsearchSpec{
			ManagementState: esv1.ManagementStateManaged,
		},
	}
	updated := esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: esv1.ElasticsearchSpec{
			ManagementState: esv1.ManagementStateUnmanaged,
		},
	}
	toDelete := esv1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []esv1.Elasticsearch{toUpdate, toDelete}
	desired := []esv1.Elasticsearch{updated, toCreate}

	inv := ForElasticsearches(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, esv1.ManagementStateUnmanaged, inv.Update[0].Spec.ManagementState)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestElasticsearchInventoryWithSameNameInstances(t *testing.T) {
	create := []esv1.Elasticsearch{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	inv := ForElasticsearches([]esv1.Elasticsearch{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
