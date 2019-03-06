package v1

import log "github.com/sirupsen/logrus"

// Logger returns a logger filled with context-related fields, such as Name and Namespace
func (j *Jaeger) Logger() *log.Entry {
	return log.WithFields(log.Fields{
		"instance":  j.Name,
		"namespace": j.Namespace,
	})
}
