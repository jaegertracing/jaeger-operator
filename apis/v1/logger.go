package v1

import (
	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Logger returns a logger filled with context-related fields, such as Name and Namespace
func (j *Jaeger) Logger() logr.Logger {
	return logf.Log.WithValues(
		"instance", j.Name,
		"namespace", j.Namespace,
	)
}
