package stub

import (
	"context"

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

		// we store back the changed CR, so that what is stored reflects what is being used
		if err := sdk.Update(o); err != nil {
			logrus.Errorf("failed to update %v", o)
			return err
		}
	}
	return nil
}
