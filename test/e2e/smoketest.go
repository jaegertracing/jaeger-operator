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

// This version is for the all-in-one image, where query and collector use the same pod
func AllInOneSmokeTest(allInOnePodPrefix, allInOnePodImageName, serviceName string, interval, timeout time.Duration) {
	queryPort := randomPortNumber()
	collectorPort := randomPortNumber()
	ports := []string{queryPort + ":16686", collectorPort + ":14268"}
	portForw, closeChan := CreatePortForward(namespace, allInOnePodPrefix, allInOnePodImageName, ports, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	apiTracesEndpoint := fmt.Sprintf("http://localhost:%s/api/traces", queryPort)
	collectorEndpoint := fmt.Sprintf("http://localhost:%s/api/traces", collectorPort)
	ExecuteSmokeTest(apiTracesEndpoint, collectorEndpoint, serviceName, interval, timeout)
}

// Call this version if query and collector are in separate pods
func ProductionSmokeTest(queryPodPrefix, queryPodImageName, collectorPodPrefix, collectorPodImageName, serviceName string, interval, timeout time.Duration) {
	queryPort := randomPortNumber()
	queryPorts := []string{queryPort + ":16686"}
	portForw, closeChan := CreatePortForward(namespace, queryPodPrefix, queryPodImageName, queryPorts, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	collectorPort := randomPortNumber()
	collectorPorts := []string{collectorPort + ":14268"}
	portForwColl, closeChanColl := CreatePortForward(namespace, collectorPodPrefix, collectorPodImageName, collectorPorts, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)

	apiTracesEndpoint := fmt.Sprintf("http://localhost:%s/api/traces", queryPort)
	collectorEndpoint := fmt.Sprintf("http://localhost:%s/api/traces", collectorPort)
	ExecuteSmokeTest(apiTracesEndpoint, collectorEndpoint, serviceName, interval, timeout)
}

func ExecuteSmokeTest(apiTracesEndpoint, collectorEndpoint, serviceName string, interval time.Duration, duration time.Duration) {
	cfg := config.Configuration{
		Reporter: &config.ReporterConfig{CollectorEndpoint: collectorEndpoint},
		Sampler: &config.SamplerConfig{Type: "const", Param: 1},
		ServiceName: serviceName,
	}
	tracer, closer, err := cfg.NewTracer()
	require.NoError(t, err, "Failed to create tracer in SmokeTest")

	tStr := time.Now().Format(time.RFC3339Nano)
	tracer.StartSpan("SmokeTest").
		SetTag("time-RFC3339Nano", tStr).
		Finish()
	closer.Close()

	err = wait.Poll(interval, timeout, func() (done bool, err error) {
		c := http.Client{Timeout: time.Second}
		req, err := http.NewRequest(http.MethodGet, apiTracesEndpoint+"?service="+serviceName, nil)
		if err != nil {
			return false, err
		}
		resp, err := c.Do(req)
		if err != nil {
			return false, nil
		}
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
