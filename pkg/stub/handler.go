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
			// aparently, everything created by the CR should cascade the deletion to the
			// resources it created, so, no need to do anything
			// on a next version, we could think about cleaning up the data pertaining to
			// us at the storage level, but not for now
			return nil
		}

		ctrl := controller.NewController(ctx, o)

		objs := ctrl.Create()
		for _, obj := range objs {
			err := sdk.Create(obj)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				logrus.Errorf("failed to create %v", obj)
				return err
			}

			if err == nil {
				logrus.Infof("Created '%v'", o.Name)
			}
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
