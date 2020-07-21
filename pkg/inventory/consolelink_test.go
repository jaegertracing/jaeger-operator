package inventory

import (
	"testing"

	osconsolev1 "github.com/openshift/api/console/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConsoleLinkInventory(t *testing.T) {
	toCreate := osconsolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := osconsolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: osconsolev1.ConsoleLinkSpec{
			Link: osconsolev1.Link{
				Href: "https://onehost",
			},
		},
	}
	updated := osconsolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: osconsolev1.ConsoleLinkSpec{
			Link: osconsolev1.Link{
				Href: "https://otherhost",
			},
		},
	}
	toDelete := osconsolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []osconsolev1.ConsoleLink{toUpdate, toDelete}
	desired := []osconsolev1.ConsoleLink{updated, toCreate}

	inv := ForConsoleLinks(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "https://otherhost", inv.Update[0].Spec.Link.Href)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
