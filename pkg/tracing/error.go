package tracing

import (
	"go.opentelemetry.io/otel/codes"
	apitrace "go.opentelemetry.io/otel/trace"
)

// HandleError sets the span to an error state, storing it s cause in an attribute
func HandleError(err error, span apitrace.Span) error {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
