package v1

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type captureSink struct {
	keysAndValues []interface{}
}

func (s *captureSink) Init(info logr.RuntimeInfo)                                   {}
func (s *captureSink) Enabled(level int) bool                                       { return true }
func (s *captureSink) Info(level int, msg string, keysAndValues ...interface{})     {}
func (s *captureSink) Error(err error, msg string, keysAndValues ...interface{})    {}
func (s *captureSink) WithValues(keysAndValues ...interface{}) logr.LogSink         { return &captureSink{keysAndValues: keysAndValues} }
func (s *captureSink) WithName(name string) logr.LogSink                            { return s }

func TestJaegerLogger(t *testing.T) {
	original := logf.Log
	defer logf.SetLogger(original)

	sink := &captureSink{}
	logf.SetLogger(logr.New(sink))

	j := &Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "test-namespace",
		},
	}

	logger := j.Logger()
	captured, ok := logger.GetSink().(*captureSink)
	assert.True(t, ok, "expected *captureSink")
	assert.Equal(t, []interface{}{"instance", "test-instance", "namespace", "test-namespace"}, captured.keysAndValues)
}

func TestJaegerLoggerReturnsNonNil(t *testing.T) {
	original := logf.Log
	defer logf.SetLogger(original)

	logf.SetLogger(logr.New(&captureSink{}))

	j := &Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-instance",
			Namespace: "some-namespace",
		},
	}
	logger := j.Logger()
	assert.NotNil(t, logger.GetSink())
}
