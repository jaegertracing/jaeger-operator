package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

// Plugin is a dummy plugin which only implements GetServices with dummy output for e2e testing plugins
type Plugin struct{}

func main() {
	// Parse command line options
	var configPath string
	flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
	flag.Parse()

	logger := hclog.New(&hclog.LoggerOptions{
		Name:       "dummy",
		Level:      hclog.Info,
		JSONFormat: true,
	})

	// Parse plugin config with tokens etc.
	logger.Warn("Config path", "path", configPath)

	// #nosec   G304: Potential file inclusion via variable
	_, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Error("Reading config failed", "err", err.Error())
		os.Exit(1)
	}

	grpc.Serve(&Plugin{})
}

// SpanReader implements spanstore.Reader
func (p *Plugin) SpanReader() spanstore.Reader { return p }

// GetServices returns hard coded results for e2e
func (p *Plugin) GetServices(ctx context.Context) ([]string, error) {
	return []string{"dummy1", "dummy2"}, nil
}

// GetTrace is not implemented
func (p *Plugin) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	return nil, spanstore.ErrTraceNotFound
}

// GetOperations is not implemented
func (p *Plugin) GetOperations(ctx context.Context, q spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	return nil, nil
}

// FindTraces is not implemented
func (p *Plugin) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}

// FindTraceIDs is not implemented
func (p *Plugin) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}

// Assert that we implement the right interface
var _ spanstore.Reader = &Plugin{}

// DependencyReader implements dependencystore.Reader
func (p *Plugin) DependencyReader() dependencystore.Reader { return p }

// GetDependencies is not implemented
func (p *Plugin) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}

// We satisfy the dependencystore.Reader interface
var _ dependencystore.Reader = &Plugin{}

// SpanWriter implements spanstore.Writer
func (p *Plugin) SpanWriter() spanstore.Writer { return p }

// WriteSpan is not implemented
func (p *Plugin) WriteSpan(span *model.Span) error { return nil }

// Assert that we implement the upstream interface
var _ spanstore.Writer = &Plugin{}
