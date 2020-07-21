package controller

import (
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	osimagev1 "github.com/openshift/api/image/v1"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	if err := routev1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := consolev1.Install(m.GetScheme()); err != nil {
		return err
	}

	// Registry just the ImageStream - adding osimagev1.AddToScheme(..) causes
	// the SecretList to be registered again, which resulted in
	// https://github.com/kubernetes-sigs/controller-runtime/issues/362
	var SchemeBuilder = &scheme.Builder{GroupVersion: osimagev1.SchemeGroupVersion}
	SchemeBuilder.Register(&osimagev1.ImageStream{})
	if err := SchemeBuilder.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
