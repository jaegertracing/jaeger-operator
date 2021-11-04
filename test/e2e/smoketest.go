package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go/config"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// AllInOneSmokeTest is for the all-in-one image, where query and collector use the same pod
func AllInOneSmokeTest(jaegerInstanceName string) {
	allInOneImageName := "all-in-one"
	ports := []string{"0:16686", "0:14268"}
	portForw, closeChan := CreatePortForward(namespace, jaegerInstanceName, allInOneImageName, ports, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)
	forwardedPorts, err := portForw.GetPorts()
	require.NoError(t, err)
	queryPort := forwardedPorts[0].Local
	collectorPort := forwardedPorts[1].Local

	// Use ingress for k8s or on OpenShift if we have an insecure route
	var apiTracesEndpoint string
	insecureEndpoint := hasInsecureEndpoint(jaegerInstanceName, namespace)
	if insecureEndpoint {
		apiTracesEndpoint = getQueryURL(jaegerInstanceName, namespace, "%s/api/traces")
	} else {
		apiTracesEndpoint = fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	}
	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint, insecureEndpoint)
}

// ProductionSmokeTest should be used if query and collector are in separate pods
func ProductionSmokeTest(resourceName string) {
	productionSmokeTest(resourceName, namespace)
}

// ProductionSmokeTestWithNamespace is the same as ProductionSmokeTest but for when you can't use the default namespace
func ProductionSmokeTestWithNamespace(resourceName, smokeTestNamespace string) {
	productionSmokeTest(resourceName, smokeTestNamespace)
}

func productionSmokeTest(jaegerInstanceName, smokeTestNamespace string) {
	queryPodImageName := "jaeger-query"
	collectorPodImageName := "collector"
	queryPodPrefix := jaegerInstanceName + "-query"
	collectorPodPrefix := jaegerInstanceName + "-collector"

	// Use ingress for k8s or on OpenShift if we have an insecure route
	var apiTracesEndpoint string
	insecureEndpoint := hasInsecureEndpoint(jaegerInstanceName, smokeTestNamespace)
	if insecureEndpoint {
		apiTracesEndpoint = getQueryURL(jaegerInstanceName, smokeTestNamespace, "%s/api/traces")
	} else {
		queryPorts := []string{"0:16686"}
		portForw, closeChan := CreatePortForward(smokeTestNamespace, queryPodPrefix, queryPodImageName, queryPorts, fw.KubeConfig)
		defer portForw.Close()
		defer close(closeChan)
		forwardedQueryPorts, err := portForw.GetPorts()
		require.NoError(t, err)
		queryPort := forwardedQueryPorts[0].Local
		apiTracesEndpoint = fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	}

	collectorPorts := []string{"0:14268"}
	portForwColl, closeChanColl := CreatePortForward(smokeTestNamespace, collectorPodPrefix, collectorPodImageName, collectorPorts, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)
	forwardedCollectorPorts, err := portForwColl.GetPorts()
	require.NoError(t, err)
	collectorPort := forwardedCollectorPorts[0].Local

	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint, insecureEndpoint)
}

func hasInsecureEndpoint(jaegerInstanceName, jaegerInstanceNamespace string) bool {
	if !isOpenShift(t) {
		return true
	}

	jaeger := getJaegerInstance(jaegerInstanceName, jaegerInstanceNamespace)
	if jaeger.Spec.Ingress.Security == v1.IngressSecurityNoneExplicit || jaeger.Spec.Ingress.Security == v1.IngressSecurityNone {
		return true
	}
	return false
}

func executeSmokeTest(apiTracesEndpoint, collectorEndpoint string, hasInsecureEndpoint bool) {
	serviceName := "smoketest" + strconv.Itoa(time.Now().Nanosecond())
	cfg := config.Configuration{
		Reporter:    &config.ReporterConfig{CollectorEndpoint: collectorEndpoint},
		Sampler:     &config.SamplerConfig{Type: "const", Param: 1},
		ServiceName: serviceName,
	}
	tracer, closer, err := cfg.NewTracer()
	require.NoError(t, err, "Failed to create tracer in SmokeTest")

	tStr := time.Now().Format(time.RFC3339Nano)
	tracer.StartSpan("SmokeTest").
		SetTag("time-RFC3339Nano", tStr).
		Finish()
	closer.Close()

	transport := &http.Transport{}
	if hasInsecureEndpoint {
		insecure := true
		// #nosec  G402: TLS InsecureSkipVerify set true
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	}
	tracesEndpoint := apiTracesEndpoint + "?service=" + serviceName
	logrus.Infof("Using traces URL %s", tracesEndpoint)

	c := http.Client{Timeout: 3 * time.Second, Transport: transport}

	resp := &resp{}
	err = WaitForHTTPResponse(c, http.MethodGet, tracesEndpoint, &resp)
	require.NoError(t, err, "Failed waiting for expected content")
}
