package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// NewJaeger returns a new Jaeger instance with the given name
func NewJaeger(name string) *Jaeger {
	return &Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
