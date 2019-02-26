package controller

import (
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	esv1alpha1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1alpha1"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	if err := routev1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	// TODO temporary fix https://github.com/jaegertracing/jaeger-operator/issues/206
	gv := schema.GroupVersion{Group: "logging.openshift.io", Version: "v1alpha1"}
	m.GetScheme().AddKnownTypes(gv, &esv1alpha1.Elasticsearch{})
	m.GetScheme().AddKnownTypes(gv, &esv1alpha1.ElasticsearchList{})

	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
