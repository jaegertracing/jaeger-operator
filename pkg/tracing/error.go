package tracing

import (
	"go.opentelemetry.io/otel/api/key"
	apitrace "go.opentelemetry.io/otel/api/trace"

	"google.golang.org/grpc/codes"
)

// HandleError sets the span to an error state, storing it s cause in an attribute
func HandleError(err error, span apitrace.Span) error {
	span.SetAttribute(key.String("error", err.Error()))
	span.SetStatus(codes.Internal)
	return err
}
