package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

func TestElasticsearchInventory(t *testing.T) {
	toCreate := esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: esv1alpha1.ElasticsearchSpec{
			ManagementState: esv1alpha1.ManagementStateManaged,
		},
	}
	updated := esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: esv1alpha1.ElasticsearchSpec{
			ManagementState: esv1alpha1.ManagementStateUnmanaged,
		},
	}
	toDelete := esv1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []esv1alpha1.Elasticsearch{toUpdate, toDelete}
	desired := []esv1alpha1.Elasticsearch{updated, toCreate}

	inv := ForElasticsearches(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, esv1alpha1.ManagementStateUnmanaged, inv.Update[0].Spec.ManagementState)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
