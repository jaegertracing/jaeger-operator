package e2e

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	osv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/require"
	"github.com/uber/jaeger-client-go/config"
	"k8s.io/apimachinery/pkg/util/wait"

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

	var apiTracesEndpoint string
	route, hasInsecureRoute := getInsecureRoute(jaegerInstanceName)
	if hasInsecureRoute {
		apiTracesEndpoint = fmt.Sprintf("https://%s/api/traces", route.Spec.Host)
	} else {
		apiTracesEndpoint = fmt.Sprintf("http://localhost:%d/api/traces", queryPort)
	}
	collectorEndpoint := fmt.Sprintf("http://localhost:%d/api/traces", collectorPort)
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint, hasInsecureRoute)
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
	collectorPodImageName := "jaeger-collector"
	queryPodPrefix := jaegerInstanceName + "-query"
	collectorPodPrefix := jaegerInstanceName + "-collector"

	var apiTracesEndpoint string
	route, hasInsecureRoute := getInsecureRoute(jaegerInstanceName)
	if hasInsecureRoute {
		apiTracesEndpoint = fmt.Sprintf("https://%s/api/traces", route.Spec.Host)
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
	executeSmokeTest(apiTracesEndpoint, collectorEndpoint, hasInsecureRoute)
}

func getInsecureRoute(jaegerInstanceName string) (*osv1.Route, bool) {
	var route *osv1.Route
	hasInsecureRoute := false
	if isOpenShift(t) {
		route = findRoute(t, fw, jaegerInstanceName)
		if route != nil {
			jaeger := getJaegerInstance(jaegerInstanceName, namespace)
			if jaeger.Spec.Ingress.Security == v1.IngressSecurityNoneExplicit || jaeger.Spec.Ingress.Security == v1.IngressSecurityNone {
				hasInsecureRoute = true
			}
		}
	}

	return route, hasInsecureRoute
}

func executeSmokeTest(apiTracesEndpoint, collectorEndpoint string, hasInsecureRoute bool) {
	logrus.Infof("Using Query endpoint %s", apiTracesEndpoint)
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

	transport := &http.Transport{}
	if hasInsecureRoute {
		insecure := true
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	}
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		c := http.Client{Timeout: 3 * time.Second, Transport: transport}
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
