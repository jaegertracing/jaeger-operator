package status

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestPodsArePending(t *testing.T) {
	jaeger := v1alpha1.Jaeger{}

	pod := v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
	}
	objs := []runtime.Object{
		&pod,
	}

	cl := fake.NewFakeClient(objs...)
	s := scraper{ksClient: cl}
	jaeger = s.scrape(jaeger)
	assert.Equal(t, 0, jaeger.Status.CollectorQueueLength)
	assert.Equal(t, 0, jaeger.Status.CollectorSpansDropped)
	assert.Equal(t, 0, jaeger.Status.CollectorSpansReceived)
	assert.Equal(t, 0, jaeger.Status.CollectorTracesReceived)
}

func TestPodsAreRunning(t *testing.T) {
	assertMetricsAreCollected(t, v1alpha1.Jaeger{})
}

func TestOldStatusIsReplaced(t *testing.T) {
	assertMetricsAreCollected(t, v1alpha1.Jaeger{
		Status: v1alpha1.JaegerStatus{
			CollectorQueueLength:    1000,
			CollectorSpansDropped:   2000,
			CollectorSpansReceived:  3000,
			CollectorTracesReceived: 4000,
		},
	})
}

func TestAllInOnePodsAreRunning(t *testing.T) {
	jaeger := v1alpha1.Jaeger{
		Spec: v1alpha1.JaegerSpec{
			Strategy: "allInOne",
		},
	}

	assertMetricsAreCollected(t, jaeger)
}

func TestNonNumericMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`jaeger_collector_queue_length{host="d096132db661"} N`))
	}))
	// Close the server when test finishes
	defer server.Close()
	assertZeroMetricsOnFailure(t, server)
}

func TestUnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(404)
	}))
	// Close the server when test finishes
	defer server.Close()

	assertZeroMetricsOnFailure(t, server)
}

func assertMetricsAreCollected(t *testing.T, jaeger v1alpha1.Jaeger) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(response))
	}))
	// Close the server when test finishes
	defer server.Close()

	port, err := strconv.Atoi(server.URL[strings.LastIndex(server.URL, ":")+1:])
	assert.NoError(t, err)

	pod := v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			PodIP: "127.0.0.1",
		},
	}
	objs := []runtime.Object{
		&pod,
	}

	cl := fake.NewFakeClient(objs...)
	s := scraper{ksClient: cl, hClient: *server.Client(), targetPort: port}
	jaeger = s.scrape(jaeger)

	assert.Equal(t, 2, jaeger.Status.CollectorQueueLength)
	assert.Equal(t, 1, jaeger.Status.CollectorSpansDropped)
	assert.Equal(t, 10, jaeger.Status.CollectorSpansReceived)
	assert.Equal(t, 20, jaeger.Status.CollectorTracesReceived)
}

func assertZeroMetricsOnFailure(t *testing.T, server *httptest.Server) {
	jaeger := v1alpha1.Jaeger{}

	port, err := strconv.Atoi(server.URL[strings.LastIndex(server.URL, ":")+1:])
	assert.NoError(t, err)

	pod := v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			PodIP: "127.0.0.1",
		},
	}
	objs := []runtime.Object{
		&pod,
	}

	cl := fake.NewFakeClient(objs...)
	s := scraper{ksClient: cl, hClient: *server.Client(), targetPort: port}
	jaeger = s.scrape(jaeger)

	assert.Equal(t, 0, jaeger.Status.CollectorQueueLength)
	assert.Equal(t, 0, jaeger.Status.CollectorSpansDropped)
	assert.Equal(t, 0, jaeger.Status.CollectorSpansReceived)
	assert.Equal(t, 0, jaeger.Status.CollectorTracesReceived)
}

type fakeHClient struct {
	response http.Response
	err      error
}

func (f *fakeHClient) Do(req *http.Request) (*http.Response, error) {
	return &f.response, f.err
}

const (
	response = `# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 1.9056e-05
go_gc_duration_seconds{quantile="0.25"} 7.6033e-05
go_gc_duration_seconds{quantile="0.5"} 0.000137386
go_gc_duration_seconds{quantile="0.75"} 0.000350048
go_gc_duration_seconds{quantile="1"} 0.002085805
go_gc_duration_seconds_sum 0.018923647
go_gc_duration_seconds_count 56
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 138
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 1.4895416e+07
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 4.71136712e+08
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.474023e+06
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 6.150701e+06
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 2.387968e+06
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 1.4895416e+07
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 4.9709056e+07
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 1.622016e+07
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 63359
# HELP go_memstats_heap_released_bytes_total Total number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes_total counter
go_memstats_heap_released_bytes_total 4.3352064e+07
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 6.5929216e+07
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.551452354404261e+09
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 0
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 6.21406e+06
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 3456
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 16384
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 136344
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 229376
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 2.1155808e+07
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 543505
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 1.179648e+06
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 1.179648e+06
# HELP go_memstats_sys_bytes Number of bytes obtained by system. Sum of all system allocations.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 7.176012e+07
# HELP jaeger_agent_collector_proxy_total collector-proxy
# TYPE jaeger_agent_collector_proxy_total counter
jaeger_agent_collector_proxy_total{endpoint="baggage",protocol="tchannel",result="err"} 0
jaeger_agent_collector_proxy_total{endpoint="baggage",protocol="tchannel",result="ok"} 0
jaeger_agent_collector_proxy_total{endpoint="sampling",protocol="tchannel",result="err"} 0
jaeger_agent_collector_proxy_total{endpoint="sampling",protocol="tchannel",result="ok"} 0
# HELP jaeger_agent_reporter_batch_size batch_size
# TYPE jaeger_agent_reporter_batch_size gauge
jaeger_agent_reporter_batch_size{format="jaeger",protocol="tchannel"} 1
jaeger_agent_reporter_batch_size{format="zipkin",protocol="tchannel"} 0
# HELP jaeger_agent_reporter_batches_failures_total batches.failures
# TYPE jaeger_agent_reporter_batches_failures_total counter
jaeger_agent_reporter_batches_failures_total{format="jaeger",protocol="tchannel"} 0
jaeger_agent_reporter_batches_failures_total{format="zipkin",protocol="tchannel"} 0
# HELP jaeger_agent_reporter_batches_submitted_total batches.submitted
# TYPE jaeger_agent_reporter_batches_submitted_total counter
jaeger_agent_reporter_batches_submitted_total{format="jaeger",protocol="tchannel"} 9
jaeger_agent_reporter_batches_submitted_total{format="zipkin",protocol="tchannel"} 0
# HELP jaeger_agent_reporter_spans_failures_total spans.failures
# TYPE jaeger_agent_reporter_spans_failures_total counter
jaeger_agent_reporter_spans_failures_total{format="jaeger",protocol="tchannel"} 0
jaeger_agent_reporter_spans_failures_total{format="zipkin",protocol="tchannel"} 0
# HELP jaeger_agent_reporter_spans_submitted_total spans.submitted
# TYPE jaeger_agent_reporter_spans_submitted_total counter
jaeger_agent_reporter_spans_submitted_total{format="jaeger",protocol="tchannel"} 10
jaeger_agent_reporter_spans_submitted_total{format="zipkin",protocol="tchannel"} 0
# HELP jaeger_collector_batch_size batch-size
# TYPE jaeger_collector_batch_size gauge
jaeger_collector_batch_size{host="simplest-7d6c6ff69-8l2v7"} 1
# HELP jaeger_collector_in_queue_latency in-queue-latency
# TYPE jaeger_collector_in_queue_latency histogram
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.005"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.01"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.025"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.05"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.1"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.25"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.5"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="1"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="2.5"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="5"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="10"} 10
jaeger_collector_in_queue_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="+Inf"} 10
jaeger_collector_in_queue_latency_sum{host="simplest-7d6c6ff69-8l2v7"} 0.002283163
jaeger_collector_in_queue_latency_count{host="simplest-7d6c6ff69-8l2v7"} 10
# HELP jaeger_collector_queue_length queue-length
# TYPE jaeger_collector_queue_length gauge
jaeger_collector_queue_length{host="simplest-7d6c6ff69-8l2v7"} 2
# HELP jaeger_collector_save_latency save-latency
# TYPE jaeger_collector_save_latency histogram
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.005"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.01"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.025"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.05"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.1"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.25"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="0.5"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="1"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="2.5"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="5"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="10"} 10
jaeger_collector_save_latency_bucket{host="simplest-7d6c6ff69-8l2v7",le="+Inf"} 10
jaeger_collector_save_latency_sum{host="simplest-7d6c6ff69-8l2v7"} 0.000547021
jaeger_collector_save_latency_count{host="simplest-7d6c6ff69-8l2v7"} 10
# HELP jaeger_collector_spans_dropped_total spans.dropped
# TYPE jaeger_collector_spans_dropped_total counter
jaeger_collector_spans_dropped_total{host="simplest-7d6c6ff69-8l2v7"} 1
# HELP jaeger_collector_spans_received_total received
# TYPE jaeger_collector_spans_received_total counter
jaeger_collector_spans_received_total{debug="false",format="jaeger",svc="jaeger-query"} 10
jaeger_collector_spans_received_total{debug="false",format="jaeger",svc="other-services"} 0
jaeger_collector_spans_received_total{debug="false",format="unknown",svc="other-services"} 0
jaeger_collector_spans_received_total{debug="false",format="zipkin",svc="other-services"} 0
jaeger_collector_spans_received_total{debug="true",format="jaeger",svc="other-services"} 0
jaeger_collector_spans_received_total{debug="true",format="unknown",svc="other-services"} 0
jaeger_collector_spans_received_total{debug="true",format="zipkin",svc="other-services"} 0
# HELP jaeger_collector_spans_rejected_total rejected
# TYPE jaeger_collector_spans_rejected_total counter
jaeger_collector_spans_rejected_total{debug="false",format="jaeger",svc="other-services"} 0
jaeger_collector_spans_rejected_total{debug="false",format="unknown",svc="other-services"} 0
jaeger_collector_spans_rejected_total{debug="false",format="zipkin",svc="other-services"} 0
jaeger_collector_spans_rejected_total{debug="true",format="jaeger",svc="other-services"} 0
jaeger_collector_spans_rejected_total{debug="true",format="unknown",svc="other-services"} 0
jaeger_collector_spans_rejected_total{debug="true",format="zipkin",svc="other-services"} 0
# HELP jaeger_collector_spans_saved_by_svc_total saved-by-svc
# TYPE jaeger_collector_spans_saved_by_svc_total counter
jaeger_collector_spans_saved_by_svc_total{debug="false",result="err",svc="other-services"} 0
jaeger_collector_spans_saved_by_svc_total{debug="false",result="ok",svc="jaeger-query"} 10
jaeger_collector_spans_saved_by_svc_total{debug="false",result="ok",svc="other-services"} 0
jaeger_collector_spans_saved_by_svc_total{debug="true",result="err",svc="other-services"} 0
jaeger_collector_spans_saved_by_svc_total{debug="true",result="ok",svc="other-services"} 0
# HELP jaeger_collector_spans_serviceNames spans.serviceNames
# TYPE jaeger_collector_spans_serviceNames gauge
jaeger_collector_spans_serviceNames{host="simplest-7d6c6ff69-8l2v7"} 0
# HELP jaeger_collector_traces_received_total received
# TYPE jaeger_collector_traces_received_total counter
jaeger_collector_traces_received_total{debug="false",format="jaeger",svc="jaeger-query"} 20
jaeger_collector_traces_received_total{debug="false",format="jaeger",svc="other-services"} 0
jaeger_collector_traces_received_total{debug="false",format="unknown",svc="other-services"} 0
jaeger_collector_traces_received_total{debug="false",format="zipkin",svc="other-services"} 0
jaeger_collector_traces_received_total{debug="true",format="jaeger",svc="other-services"} 0
jaeger_collector_traces_received_total{debug="true",format="unknown",svc="other-services"} 0
jaeger_collector_traces_received_total{debug="true",format="zipkin",svc="other-services"} 0
# HELP jaeger_collector_traces_rejected_total rejected
# TYPE jaeger_collector_traces_rejected_total counter
jaeger_collector_traces_rejected_total{debug="false",format="jaeger",svc="other-services"} 0
jaeger_collector_traces_rejected_total{debug="false",format="unknown",svc="other-services"} 0
jaeger_collector_traces_rejected_total{debug="false",format="zipkin",svc="other-services"} 0
jaeger_collector_traces_rejected_total{debug="true",format="jaeger",svc="other-services"} 0
jaeger_collector_traces_rejected_total{debug="true",format="unknown",svc="other-services"} 0
jaeger_collector_traces_rejected_total{debug="true",format="zipkin",svc="other-services"} 0
# HELP jaeger_collector_traces_saved_by_svc_total saved-by-svc
# TYPE jaeger_collector_traces_saved_by_svc_total counter
jaeger_collector_traces_saved_by_svc_total{debug="false",result="err",svc="other-services"} 0
jaeger_collector_traces_saved_by_svc_total{debug="false",result="ok",svc="jaeger-query"} 10
jaeger_collector_traces_saved_by_svc_total{debug="false",result="ok",svc="other-services"} 0
jaeger_collector_traces_saved_by_svc_total{debug="true",result="err",svc="other-services"} 0
jaeger_collector_traces_saved_by_svc_total{debug="true",result="ok",svc="other-services"} 0
# HELP jaeger_http_server_errors_total http-server.errors
# TYPE jaeger_http_server_errors_total counter
jaeger_http_server_errors_total{source="all",status="4xx"} 0
jaeger_http_server_errors_total{source="collector-proxy",status="5xx"} 0
jaeger_http_server_errors_total{source="thrift",status="5xx"} 0
jaeger_http_server_errors_total{source="write",status="5xx"} 0
# HELP jaeger_http_server_requests_total http-server.requests
# TYPE jaeger_http_server_requests_total counter
jaeger_http_server_requests_total{type="baggage"} 0
jaeger_http_server_requests_total{type="sampling"} 0
jaeger_http_server_requests_total{type="sampling-legacy"} 0
# HELP jaeger_query_latency latency
# TYPE jaeger_query_latency histogram
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.005"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.01"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.025"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.05"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.1"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.25"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="0.5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="1"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="2.5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="10"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="err",le="+Inf"} 0
jaeger_query_latency_sum{operation="find_trace_ids",result="err"} 0
jaeger_query_latency_count{operation="find_trace_ids",result="err"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.005"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.01"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.025"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.05"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.1"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.25"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="0.5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="1"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="2.5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="5"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="10"} 0
jaeger_query_latency_bucket{operation="find_trace_ids",result="ok",le="+Inf"} 0
jaeger_query_latency_sum{operation="find_trace_ids",result="ok"} 0
jaeger_query_latency_count{operation="find_trace_ids",result="ok"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.005"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.01"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.025"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.05"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.1"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.25"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="0.5"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="1"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="2.5"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="5"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="10"} 0
jaeger_query_latency_bucket{operation="find_traces",result="err",le="+Inf"} 0
jaeger_query_latency_sum{operation="find_traces",result="err"} 0
jaeger_query_latency_count{operation="find_traces",result="err"} 0
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.005"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.01"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.025"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.05"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.1"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.25"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="0.5"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="1"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="2.5"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="5"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="10"} 4
jaeger_query_latency_bucket{operation="find_traces",result="ok",le="+Inf"} 4
jaeger_query_latency_sum{operation="find_traces",result="ok"} 7.2977e-05
jaeger_query_latency_count{operation="find_traces",result="ok"} 4
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.005"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.01"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.025"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.05"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.1"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.25"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="0.5"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="1"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="2.5"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="5"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="10"} 0
jaeger_query_latency_bucket{operation="get_operations",result="err",le="+Inf"} 0
jaeger_query_latency_sum{operation="get_operations",result="err"} 0
jaeger_query_latency_count{operation="get_operations",result="err"} 0
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.005"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.01"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.025"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.05"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.1"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.25"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="0.5"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="1"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="2.5"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="5"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="10"} 2
jaeger_query_latency_bucket{operation="get_operations",result="ok",le="+Inf"} 2
jaeger_query_latency_sum{operation="get_operations",result="ok"} 6.1740000000000005e-06
jaeger_query_latency_count{operation="get_operations",result="ok"} 2
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.005"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.01"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.025"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.05"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.1"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.25"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="0.5"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="1"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="2.5"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="5"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="10"} 0
jaeger_query_latency_bucket{operation="get_services",result="err",le="+Inf"} 0
jaeger_query_latency_sum{operation="get_services",result="err"} 0
jaeger_query_latency_count{operation="get_services",result="err"} 0
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.005"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.01"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.025"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.05"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.1"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.25"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="0.5"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="1"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="2.5"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="5"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="10"} 4
jaeger_query_latency_bucket{operation="get_services",result="ok",le="+Inf"} 4
jaeger_query_latency_sum{operation="get_services",result="ok"} 6.626000000000001e-06
jaeger_query_latency_count{operation="get_services",result="ok"} 4
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.005"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.01"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.025"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.05"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.1"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.25"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="0.5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="1"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="2.5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="10"} 0
jaeger_query_latency_bucket{operation="get_trace",result="err",le="+Inf"} 0
jaeger_query_latency_sum{operation="get_trace",result="err"} 0
jaeger_query_latency_count{operation="get_trace",result="err"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.005"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.01"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.025"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.05"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.1"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.25"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="0.5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="1"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="2.5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="5"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="10"} 0
jaeger_query_latency_bucket{operation="get_trace",result="ok",le="+Inf"} 0
jaeger_query_latency_sum{operation="get_trace",result="ok"} 0
jaeger_query_latency_count{operation="get_trace",result="ok"} 0
# HELP jaeger_query_requests_total requests
# TYPE jaeger_query_requests_total counter
jaeger_query_requests_total{operation="find_trace_ids",result="err"} 0
jaeger_query_requests_total{operation="find_trace_ids",result="ok"} 0
jaeger_query_requests_total{operation="find_traces",result="err"} 0
jaeger_query_requests_total{operation="find_traces",result="ok"} 4
jaeger_query_requests_total{operation="get_operations",result="err"} 0
jaeger_query_requests_total{operation="get_operations",result="ok"} 2
jaeger_query_requests_total{operation="get_services",result="err"} 0
jaeger_query_requests_total{operation="get_services",result="ok"} 4
jaeger_query_requests_total{operation="get_trace",result="err"} 0
jaeger_query_requests_total{operation="get_trace",result="ok"} 0
# HELP jaeger_query_responses responses
# TYPE jaeger_query_responses histogram
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.005"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.01"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.025"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.05"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.1"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.25"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="0.5"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="1"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="2.5"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="5"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="10"} 0
jaeger_query_responses_bucket{operation="find_trace_ids",le="+Inf"} 0
jaeger_query_responses_sum{operation="find_trace_ids"} 0
jaeger_query_responses_count{operation="find_trace_ids"} 0
jaeger_query_responses_bucket{operation="find_traces",le="0.005"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.01"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.025"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.05"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.1"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.25"} 4
jaeger_query_responses_bucket{operation="find_traces",le="0.5"} 4
jaeger_query_responses_bucket{operation="find_traces",le="1"} 4
jaeger_query_responses_bucket{operation="find_traces",le="2.5"} 4
jaeger_query_responses_bucket{operation="find_traces",le="5"} 4
jaeger_query_responses_bucket{operation="find_traces",le="10"} 4
jaeger_query_responses_bucket{operation="find_traces",le="+Inf"} 4
jaeger_query_responses_sum{operation="find_traces"} 2.4000000000000003e-08
jaeger_query_responses_count{operation="find_traces"} 4
jaeger_query_responses_bucket{operation="get_operations",le="0.005"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.01"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.025"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.05"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.1"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.25"} 2
jaeger_query_responses_bucket{operation="get_operations",le="0.5"} 2
jaeger_query_responses_bucket{operation="get_operations",le="1"} 2
jaeger_query_responses_bucket{operation="get_operations",le="2.5"} 2
jaeger_query_responses_bucket{operation="get_operations",le="5"} 2
jaeger_query_responses_bucket{operation="get_operations",le="10"} 2
jaeger_query_responses_bucket{operation="get_operations",le="+Inf"} 2
jaeger_query_responses_sum{operation="get_operations"} 4e-09
jaeger_query_responses_count{operation="get_operations"} 2
jaeger_query_responses_bucket{operation="get_services",le="0.005"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.01"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.025"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.05"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.1"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.25"} 4
jaeger_query_responses_bucket{operation="get_services",le="0.5"} 4
jaeger_query_responses_bucket{operation="get_services",le="1"} 4
jaeger_query_responses_bucket{operation="get_services",le="2.5"} 4
jaeger_query_responses_bucket{operation="get_services",le="5"} 4
jaeger_query_responses_bucket{operation="get_services",le="10"} 4
jaeger_query_responses_bucket{operation="get_services",le="+Inf"} 4
jaeger_query_responses_sum{operation="get_services"} 3.0000000000000004e-09
jaeger_query_responses_count{operation="get_services"} 4
jaeger_query_responses_bucket{operation="get_trace",le="0.005"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.01"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.025"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.05"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.1"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.25"} 0
jaeger_query_responses_bucket{operation="get_trace",le="0.5"} 0
jaeger_query_responses_bucket{operation="get_trace",le="1"} 0
jaeger_query_responses_bucket{operation="get_trace",le="2.5"} 0
jaeger_query_responses_bucket{operation="get_trace",le="5"} 0
jaeger_query_responses_bucket{operation="get_trace",le="10"} 0
jaeger_query_responses_bucket{operation="get_trace",le="+Inf"} 0
jaeger_query_responses_sum{operation="get_trace"} 0
jaeger_query_responses_count{operation="get_trace"} 0
# HELP jaeger_rpc_http_requests_total http_requests
# TYPE jaeger_rpc_http_requests_total counter
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services",status_code="2xx"} 4
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services",status_code="3xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services",status_code="4xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services",status_code="5xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",status_code="2xx"} 2
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",status_code="3xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",status_code="4xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",status_code="5xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/traces",status_code="2xx"} 4
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/traces",status_code="3xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/traces",status_code="4xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="/api/traces",status_code="5xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="Collector--submitBatches",status_code="2xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="Collector--submitBatches",status_code="3xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="Collector--submitBatches",status_code="4xx"} 0
jaeger_rpc_http_requests_total{component="jaeger",endpoint="Collector--submitBatches",status_code="5xx"} 0
# HELP jaeger_rpc_request_latency request_latency
# TYPE jaeger_rpc_request_latency histogram
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.005"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.01"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.025"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.05"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.1"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.25"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="0.5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="1"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="2.5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="10"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="false",le="+Inf"} 4
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/services",error="false"} 0.00042884199999999997
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/services",error="false"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.005"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.01"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.025"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.05"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.25"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="0.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="2.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="10"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services",error="true",le="+Inf"} 0
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/services",error="true"} 0
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/services",error="true"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.005"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.01"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.025"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.05"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.1"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.25"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="0.5"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="1"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="2.5"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="5"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="10"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="false",le="+Inf"} 2
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/services/-service-/operations",error="false"} 0.000314584
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/services/-service-/operations",error="false"} 2
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.005"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.01"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.025"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.05"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.25"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="0.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="2.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="10"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/services/-service-/operations",error="true",le="+Inf"} 0
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/services/-service-/operations",error="true"} 0
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/services/-service-/operations",error="true"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.005"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.01"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.025"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.05"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.1"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.25"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="0.5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="1"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="2.5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="5"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="10"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="false",le="+Inf"} 4
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/traces",error="false"} 0.002404736
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/traces",error="false"} 4
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.005"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.01"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.025"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.05"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.25"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="0.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="2.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="10"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="/api/traces",error="true",le="+Inf"} 0
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="/api/traces",error="true"} 0
jaeger_rpc_request_latency_count{component="jaeger",endpoint="/api/traces",error="true"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.005"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.01"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.025"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.05"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.1"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.25"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="0.5"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="1"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="2.5"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="5"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="10"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="false",le="+Inf"} 9
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="Collector--submitBatches",error="false"} 0.0037821
jaeger_rpc_request_latency_count{component="jaeger",endpoint="Collector--submitBatches",error="false"} 9
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.005"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.01"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.025"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.05"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.25"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="0.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="1"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="2.5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="5"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="10"} 0
jaeger_rpc_request_latency_bucket{component="jaeger",endpoint="Collector--submitBatches",error="true",le="+Inf"} 0
jaeger_rpc_request_latency_sum{component="jaeger",endpoint="Collector--submitBatches",error="true"} 0
jaeger_rpc_request_latency_count{component="jaeger",endpoint="Collector--submitBatches",error="true"} 0
# HELP jaeger_rpc_requests_total requests
# TYPE jaeger_rpc_requests_total counter
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/services",error="false"} 4
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/services",error="true"} 0
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",error="false"} 2
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/services/-service-/operations",error="true"} 0
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/traces",error="false"} 4
jaeger_rpc_requests_total{component="jaeger",endpoint="/api/traces",error="true"} 0
jaeger_rpc_requests_total{component="jaeger",endpoint="Collector--submitBatches",error="false"} 9
jaeger_rpc_requests_total{component="jaeger",endpoint="Collector--submitBatches",error="true"} 0
# HELP jaeger_thrift_udp_server_packet_size thrift.udp.server.packet_size
# TYPE jaeger_thrift_udp_server_packet_size gauge
jaeger_thrift_udp_server_packet_size{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_server_packet_size{model="jaeger",protocol="compact"} 489
jaeger_thrift_udp_server_packet_size{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_server_packets_dropped_total thrift.udp.server.packets.dropped
# TYPE jaeger_thrift_udp_server_packets_dropped_total counter
jaeger_thrift_udp_server_packets_dropped_total{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_server_packets_dropped_total{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_server_packets_dropped_total{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_server_packets_processed_total thrift.udp.server.packets.processed
# TYPE jaeger_thrift_udp_server_packets_processed_total counter
jaeger_thrift_udp_server_packets_processed_total{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_server_packets_processed_total{model="jaeger",protocol="compact"} 9
jaeger_thrift_udp_server_packets_processed_total{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_server_queue_size thrift.udp.server.queue_size
# TYPE jaeger_thrift_udp_server_queue_size gauge
jaeger_thrift_udp_server_queue_size{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_server_queue_size{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_server_queue_size{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_server_read_errors_total thrift.udp.server.read.errors
# TYPE jaeger_thrift_udp_server_read_errors_total counter
jaeger_thrift_udp_server_read_errors_total{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_server_read_errors_total{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_server_read_errors_total{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_t_processor_close_time thrift.udp.t-processor.close-time
# TYPE jaeger_thrift_udp_t_processor_close_time histogram
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.005"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.01"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.025"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.05"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.25"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="0.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="2.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="10"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="binary",le="+Inf"} 0
jaeger_thrift_udp_t_processor_close_time_sum{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_t_processor_close_time_count{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.005"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.01"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.025"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.05"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.25"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="0.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="2.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="10"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="jaeger",protocol="compact",le="+Inf"} 0
jaeger_thrift_udp_t_processor_close_time_sum{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_t_processor_close_time_count{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.005"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.01"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.025"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.05"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.25"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="0.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="1"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="2.5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="5"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="10"} 0
jaeger_thrift_udp_t_processor_close_time_bucket{model="zipkin",protocol="compact",le="+Inf"} 0
jaeger_thrift_udp_t_processor_close_time_sum{model="zipkin",protocol="compact"} 0
jaeger_thrift_udp_t_processor_close_time_count{model="zipkin",protocol="compact"} 0
# HELP jaeger_thrift_udp_t_processor_handler_errors_total thrift.udp.t-processor.handler-errors
# TYPE jaeger_thrift_udp_t_processor_handler_errors_total counter
jaeger_thrift_udp_t_processor_handler_errors_total{model="jaeger",protocol="binary"} 0
jaeger_thrift_udp_t_processor_handler_errors_total{model="jaeger",protocol="compact"} 0
jaeger_thrift_udp_t_processor_handler_errors_total{model="zipkin",protocol="compact"} 0
# HELP jaeger_tracer_baggage_restrictions_updates_total Number of times baggage restrictions were successfully updated
# TYPE jaeger_tracer_baggage_restrictions_updates_total counter
jaeger_tracer_baggage_restrictions_updates_total{result="err"} 0
jaeger_tracer_baggage_restrictions_updates_total{result="ok"} 0
# HELP jaeger_tracer_baggage_truncations_total Number of times baggage was truncated as per baggage restrictions
# TYPE jaeger_tracer_baggage_truncations_total counter
jaeger_tracer_baggage_truncations_total 0
# HELP jaeger_tracer_baggage_updates_total Number of times baggage was successfully written or updated on spans
# TYPE jaeger_tracer_baggage_updates_total counter
jaeger_tracer_baggage_updates_total{result="err"} 0
jaeger_tracer_baggage_updates_total{result="ok"} 0
# HELP jaeger_tracer_finished_spans_total Number of spans finished by this tracer
# TYPE jaeger_tracer_finished_spans_total counter
jaeger_tracer_finished_spans_total 28
# HELP jaeger_tracer_reporter_queue_length Current number of spans in the reporter queue
# TYPE jaeger_tracer_reporter_queue_length gauge
jaeger_tracer_reporter_queue_length 0
# HELP jaeger_tracer_reporter_spans_total Number of spans successfully reported
# TYPE jaeger_tracer_reporter_spans_total counter
jaeger_tracer_reporter_spans_total{result="dropped"} 0
jaeger_tracer_reporter_spans_total{result="err"} 0
jaeger_tracer_reporter_spans_total{result="ok"} 10
# HELP jaeger_tracer_sampler_queries_total Number of times the Sampler succeeded to retrieve sampling strategy
# TYPE jaeger_tracer_sampler_queries_total counter
jaeger_tracer_sampler_queries_total{result="err"} 0
jaeger_tracer_sampler_queries_total{result="ok"} 0
# HELP jaeger_tracer_sampler_updates_total Number of times the Sampler succeeded to retrieve and update sampling strategy
# TYPE jaeger_tracer_sampler_updates_total counter
jaeger_tracer_sampler_updates_total{result="err"} 0
jaeger_tracer_sampler_updates_total{result="ok"} 0
# HELP jaeger_tracer_span_context_decoding_errors_total Number of errors decoding tracing context
# TYPE jaeger_tracer_span_context_decoding_errors_total counter
jaeger_tracer_span_context_decoding_errors_total 0
# HELP jaeger_tracer_started_spans_total Number of sampled spans started by this tracer
# TYPE jaeger_tracer_started_spans_total counter
jaeger_tracer_started_spans_total{sampled="n"} 9
jaeger_tracer_started_spans_total{sampled="y"} 19
# HELP jaeger_tracer_throttled_debug_spans_total Number of times debug spans were throttled
# TYPE jaeger_tracer_throttled_debug_spans_total counter
jaeger_tracer_throttled_debug_spans_total 0
# HELP jaeger_tracer_throttler_updates_total Number of times throttler successfully updated
# TYPE jaeger_tracer_throttler_updates_total counter
jaeger_tracer_throttler_updates_total{result="err"} 0
jaeger_tracer_throttler_updates_total{result="ok"} 0
# HELP jaeger_tracer_traces_total Number of traces started by this tracer as sampled
# TYPE jaeger_tracer_traces_total counter
jaeger_tracer_traces_total{sampled="n",state="joined"} 9
jaeger_tracer_traces_total{sampled="n",state="started"} 0
jaeger_tracer_traces_total{sampled="y",state="joined"} 0
jaeger_tracer_traces_total{sampled="y",state="started"} 19
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 7.14
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 20
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 4.3470848e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.55145081056e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 1.37285632e+08
`
)
