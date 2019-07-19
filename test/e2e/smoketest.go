package e2e

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go/config"
	"k8s.io/apimachinery/pkg/util/wait"
)

// This version is for the all-in-one image, where query and collector use the same pod
func SmokeTest(queryPodPrefix, queryPodImageName, serviceName string, interval, timeout time.Duration) {
	portForw, closeChan := CreatePortForward(namespace, queryPodPrefix, queryPodImageName, []string{"16686", "14268"}, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)
	executeSmokeTest(serviceName, interval, timeout)
}

// Call this version if query and collector are in separate pods
func SmokeTestWithCollector(queryPodPrefix, queryPodImageName, collectorPodPrefix, collectorPodImageName, serviceName string, interval, timeout time.Duration) {
	portForw, closeChan := CreatePortForward(namespace, queryPodPrefix, queryPodImageName, []string{"16686"}, fw.KubeConfig)
	defer portForw.Close()
	defer close(closeChan)

	portForwColl, closeChanColl := CreatePortForward(namespace, collectorPodPrefix, collectorPodImageName, []string{"14268"}, fw.KubeConfig)
	defer portForwColl.Close()
	defer close(closeChanColl)

	executeSmokeTest(serviceName, interval, timeout)
}

func executeSmokeTest(serviceName string, interval time.Duration, duration time.Duration) {
	apiTracesEndpoint := "http://localhost:16686/api/traces"
	collectorEndpoint := "http://localhost:14268/api/traces"
	cfg := config.Configuration{
		Reporter:    &config.ReporterConfig{CollectorEndpoint: collectorEndpoint},
		Sampler:     &config.SamplerConfig{Type: "const", Param: 1},
		ServiceName: serviceName,
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		require.NoError(t, err, "Failed to create tracer in SmokeTest")
	}
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