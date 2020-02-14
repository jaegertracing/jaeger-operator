package jaeger

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/global"
	"google.golang.org/grpc/codes"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

var (
	// ErrDependencyRemoved is returned when a dependency existed but has been removed
	ErrDependencyRemoved = errors.New("dependency has been removed")
)

func (r *ReconcileJaeger) handleDependencies(ctx context.Context, str strategy.S) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "handleDependencies")
	defer span.End()

	for _, dep := range str.Dependencies() {
		err := r.handleDependency(ctx, str, dep)
		if err != nil {
			span.SetStatus(codes.Internal)
			span.SetAttribute(key.String("error", err.Error()))
			return tracing.HandleError(err, span)
		}
	}

	return nil
}

func (r *ReconcileJaeger) handleDependency(ctx context.Context, str strategy.S, dep batchv1.Job) error {
	tracer := global.TraceProvider().GetTracer(v1.ReconciliationTracer)
	ctx, span := tracer.Start(ctx, "handleDependency")
	defer span.End()

	span.SetAttributes(
		key.String("dependency.name", dep.Name),
		key.String("dependency.namespace", dep.Namespace),
	)

	err := r.client.Create(ctx, &dep)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		span.SetStatus(codes.Internal)
		span.SetAttribute(key.String("error", err.Error()))
		return tracing.HandleError(err, span)
	}

	// default to 2 minutes, in case we get a null pointer
	deadline := time.Duration(int64(120))
	if nil != dep.Spec.ActiveDeadlineSeconds {
		// we probably want to add a couple of seconds to this deadline, but for now, this should be sufficient
		deadline = time.Duration(int64(*dep.Spec.ActiveDeadlineSeconds))
	}

	seen := false
	once := &sync.Once{}
	return wait.PollImmediate(time.Second, deadline*time.Second, func() (done bool, err error) {
		batch := &batchv1.Job{}
		if err = r.client.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, batch); err != nil {
			if k8serrors.IsNotFound(err) {
				if seen {
					// we have seen this object before, but it doesn't exist anymore!
					// we don't have anything else to do here, break the poll
					log.WithFields(log.Fields{
						"namespace": dep.Namespace,
						"name":      dep.Name,
					}).Warn("Dependency has been removed.")
					span.SetStatus(codes.Internal)
					span.SetAttribute(key.String("error", ErrDependencyRemoved.Error()))
					return true, ErrDependencyRemoved
				}

				// the object might have not been created yet
				log.WithFields(log.Fields{
					"namespace": dep.Namespace,
					"name":      dep.Name,
				}).Debug("Dependency doesn't exist yet.")
				return false, nil
			}
			return false, tracing.HandleError(err, span)
		}

		seen = true
		// for now, we just assume each batch job has one pod
		if batch.Status.Succeeded != 1 {
			once.Do(func() {
				log.WithFields(log.Fields{
					"namespace": dep.Namespace,
					"name":      dep.Name,
				}).Debug("Waiting for dependency to complete")
			})
			return false, nil
		}

		return true, nil
	})
}
