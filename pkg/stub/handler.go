package stub

import (
	"context"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller"
)

// NewHandler constructs a new Jaeger operator handler
func NewHandler() sdk.Handler {
	return &Handler{}
}

// Handler holds the state for our handler
type Handler struct {
}

// Handle the event triggered by the operator
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Jaeger:
		if event.Deleted {
			logrus.Infof("Deleting '%s'", o.Name)
			return nil
		}

		// we need a name!
		if o.Name == "" {
			logrus.Infof("This Jaeger instance was created without a name. Setting it to 'my-jaeger'")
			o.Name = "my-jaeger"
		}

		// normalize the storage type
		if o.Spec.Storage.Type == "" {
			logrus.Infof("Storage type wasn't provided for the Jaeger instance '%v'. Falling back to 'memory'", o.Name)
			o.Spec.Storage.Type = "memory"
		}

		if unknownStorage(o.Spec.Storage.Type) {
			logrus.Infof(
				"The provided storage type for the Jaeger instance '%v' is unknown ('%v'). Falling back to 'memory'",
				o.Name,
				o.Spec.Storage.Type,
			)
			o.Spec.Storage.Type = "memory"
		}

		// normalize the deployment strategy
		if strings.ToLower(o.Spec.Strategy) != "production" {
			o.Spec.Strategy = "all-in-one"
		}

		// check for incompatible options
		// if the storage is `memory`, then the only possible strategy is `all-in-one`
		if strings.ToLower(o.Spec.Storage.Type) == "memory" && o.Spec.Strategy != "all-in-one" {
			logrus.Warnf(
				"No suitable storage was provided for the Jaeger instance '%v'. Falling back to all-in-one. Storage type: '%v'",
				o.Name,
				o.Spec.Storage.Type,
			)
			o.Spec.Strategy = "all-in-one"
		}

		// we store back the "fixed" CR, so that what is stored reflects what is being used
		if err := sdk.Update(o); err != nil {
			logrus.Errorf("failed to update %v", o)
			return err
		}

		ctrl := controller.NewController(ctx, o)

		objs := ctrl.Create()
		created := false
		for _, obj := range objs {
			err := sdk.Create(obj)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				logrus.Errorf("failed to create %v", obj)
				return err
			}

			if err == nil {
				created = true
			}
		}

		if created {
			logrus.Infof("Configured %v", o.Name)
		}

		objs = ctrl.Update()
		for _, obj := range objs {
			logrus.Debugf("Updating %v", obj)
			if err := sdk.Update(obj); err != nil {
				logrus.Errorf("failed to update %v", obj)
				return err
			}
		}
	}
	return nil
}

func unknownStorage(typ string) bool {
	known := []string{
		"memory",
		"kafka",
		"elasticsearch",
		"cassandra",
	}

	for _, k := range known {
		if strings.ToLower(typ) == k {
			return false
		}
	}

	return true
}
