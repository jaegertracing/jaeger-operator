package e2e

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go/config"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AllInOneSmokeTest is for the all-in-one image, where query and collector use the same pod
func AllInOneSmokeTest(resourceName string) {
	allInOneImageName := "all-in-one"
	ports := []string{"0:16686", "0:14268"}
	portForw, closeChan := CreatePortForward(namespace, resourceName, allInOneImageName, ports, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)
	forwardedPorts, err := portForw.GetPorts()
	require.NoError(t, err)
	queryPort := forwardedPorts[0].Local
	collectorPort := forwardedPorts[1].Local

	apiTracesEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint)
}

// ProductionSmokeTest should be used if query and collector are in separate pods
func ProductionSmokeTest(resourceName string) {
	productionSmokeTest(resourceName, namespace, 1, 1)
}

// ProductionSmokeTestMultiReplicas is an overloaded version of ProductionSmokeTest that offers replica parameters
func ProductionSmokeTestMultiReplicas(resourceName string, queryReplicas, collectorReplicas int) {
	productionSmokeTest(resourceName, namespace, queryReplicas, collectorReplicas)
}

// ProductionSmokeTestWithNamespace is the same as ProductionSmokeTest but for when you can't use the default namespace
func ProductionSmokeTestWithNamespace(resourceName, smokeTestNamespace string) {
	productionSmokeTest(resourceName, smokeTestNamespace, 1, 1)
}

func productionSmokeTest(resourceName, smokeTestNamespace string, queryReplicas, collectorReplicas int) {
	queryPodImageName := "jaeger-query"
	collectorPodImageName := "jaeger-collector"
	queryPodPrefix := resourceName + "-query"
	collectorPodPrefix := resourceName + "-collector"

	queryPorts := []string{"0:16686"}
	portForw, closeChan := CreatePortForwardMultiReplica(smokeTestNamespace, queryPodPrefix, queryPodImageName, queryPorts, fw.KubeConfig, queryReplicas)
	defer portForw.Close()
	defer close(closeChan)
	forwardedQueryPorts, err := portForw.GetPorts()
	require.NoError(t, err)
	queryPort := forwardedQueryPorts[0].Local

	collectorPorts := []string{"0:14268"}
	portForwColl, closeChanColl := CreatePortForwardMultiReplica(smokeTestNamespace, collectorPodPrefix, collectorPodImageName, collectorPorts, fw.KubeConfig, collectorReplicas)
	defer portForwColl.Close()
	defer close(closeChanColl)
	forwardedCollectorPorts, err := portForwColl.GetPorts()
	require.NoError(t, err)
	collectorPort := forwardedCollectorPorts[0].Local

	apiTracesEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint)
}

func executeSmokeTest(apiTracesEndpoint, collectorEndpoint string) {
	serviceName := "smoketest"
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

	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		c := http.Client{Timeout: 3 * time.Second}
		req, err := http.NewRequest(http.MethodGet, apiTracesEndpoint+"?service="+serviceName, nil)
		require.NoError(t, err)

		resp, err := c.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		if !strings.Contains(bodyString, "errors\":null") {
			return false, errors.New("query service returns errors: " + bodyString)
		}
		return strings.Contains(bodyString, tStr), nil
	})
	require.NoError(t, err, "SmokeTest failed")
}
