package e2e

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/uber/jaeger-client-go/config"
	"k8s.io/apimachinery/pkg/util/wait"
)

func SmokeTest(apiTracesEndpoint, collectorEndpoint, serviceName string, interval, timeout time.Duration) error {
	cfg := config.Configuration{
		Reporter: &config.ReporterConfig{CollectorEndpoint: collectorEndpoint},
		Sampler: &config.SamplerConfig{Type:"const", Param:1},
		ServiceName: serviceName,
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return err
	}

	tStr := time.Now().Format(time.RFC3339Nano)
	tracer.StartSpan("SmokeTest").
		SetTag("time-RFC3339Nano", tStr).
		Finish()
	closer.Close()

	return wait.Poll(interval, timeout, func() (done bool, err error) {
		c := http.Client{Timeout: time.Second}
		req, err := http.NewRequest(http.MethodGet, apiTracesEndpoint+ "?service=" + serviceName, nil)
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
}
