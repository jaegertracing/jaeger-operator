package v1

import (
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NewJaeger returns a new Jaeger instance with the given name
func NewJaeger(nsn types.NamespacedName) *Jaeger {
	return &Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				LabelOperatedBy: viper.GetString(ConfigIdentity),
			},
		},
	}
}
