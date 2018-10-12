package stub

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/controller"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
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
	case *appsv1.Deployment:
		if inject.Needed(o) {
			pods := &v1alpha1.JaegerList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Jaeger",
					APIVersion: fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version),
				},
			}
			err := sdk.List(o.GetNamespace(), pods)
			if err != nil {
				logrus.WithError(err).Error("failed to get the available Jaeger pods")
				return err
			}

			jaeger := inject.Select(o, pods)
			if jaeger != nil {
				// a suitable jaeger instance was found! let's inject a sidecar pointing to it then
				logrus.WithFields(logrus.Fields{"deployment": o.Name, "jaeger": jaeger.Name}).Info("Injecting Jaeger Agent sidecar")
				inject.Sidecar(o, jaeger)
				if err := sdk.Update(o); err != nil {
					logrus.WithField("deployment", o).Error("failed to update")
					return err
				}
			} else {
				logrus.WithField("deployment", o.Name).Info("No suitable Jaeger instances found to inject a sidecar")
			}
		}

	}
	return nil
}
