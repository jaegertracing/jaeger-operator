package status

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

type scraper struct {
	ksClient   client.Client
	hClient    http.Client
	targetPort int
}

func withClient(c client.Client) scraper {
	return scraper{ksClient: c, hClient: http.Client{Timeout: time.Second}}
}

// Scrape fills a JaegerStatus object with the current state of all Jaeger pods (currently only collectors)
func Scrape(c client.Client, jaeger v1alpha1.Jaeger) v1alpha1.Jaeger {
	s := withClient(c)
	return s.scrape(jaeger)
}

func (s *scraper) scrape(jaeger v1alpha1.Jaeger) v1alpha1.Jaeger {
	// reset the status object
	jaeger.Status = v1alpha1.JaegerStatus{}

	if strings.EqualFold(jaeger.Spec.Strategy, "allinone") {
		return s.allInOnePodsState(jaeger)
	}

	return s.collectorPodsState(jaeger)
}

func (s *scraper) allInOnePodsState(jaeger v1alpha1.Jaeger) v1alpha1.Jaeger {
	if s.targetPort == 0 {
		s.targetPort = 16686
	}

	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
		"app.kubernetes.io/component":  "all-in-one",
	})

	return s.matchingPodsState(jaeger, opts)
}

func (s *scraper) collectorPodsState(jaeger v1alpha1.Jaeger) v1alpha1.Jaeger {
	if s.targetPort == 0 {
		s.targetPort = 14268
	}

	opts := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/managed-by": "jaeger-operator",
		"app.kubernetes.io/component":  "collector",
	})

	return s.matchingPodsState(jaeger, opts)
}

func (s *scraper) matchingPodsState(jaeger v1alpha1.Jaeger, opts *client.ListOptions) v1alpha1.Jaeger {
	list := &v1.PodList{}
	if err := s.ksClient.List(context.Background(), opts, list); err != nil {
		jaeger.Logger().WithError(err).Error("failed to obtain the list of collectors")
		return jaeger
	}

	// we expect only one pod here, so, no need to think about concurrency
	for _, pod := range list.Items {
		jaeger.Logger().WithField("name", pod.Name).Debug("pod is part of the instance")

		// the pod might still be starting, or terminating... we want metrics only from running pods
		if pod.Status.Phase == v1.PodRunning {
			url := fmt.Sprintf("http://%s:%d/metrics", pod.Status.PodIP, s.targetPort)
			jaeger = s.aggregate(jaeger, url)
		}
	}

	return jaeger
}

func (s *scraper) aggregate(jaeger v1alpha1.Jaeger, url string) v1alpha1.Jaeger {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	res, err := s.hClient.Do(req)
	if err != nil {
		// TODO: decide whether this problem is serious enough to deserve a Warn/Error...
		// it should be OK to sporadically fail, or fail all the time when the operator isn't running
		// inside the cluster
		jaeger.Logger().WithField("url", url).WithError(err).Debug("failed to obtain the metrics from pod")
		return jaeger
	}

	if res.StatusCode != 200 {
		jaeger.Logger().WithField("code", res.StatusCode).WithError(err).Error("unexpected status code")
		return jaeger
	}

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		jaeger = s.extractMetric(jaeger, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		jaeger.Logger().WithField("url", url).WithError(err).Error("cannot read the response")
		return jaeger
	}

	return jaeger
}

func (s *scraper) extractMetric(jaeger v1alpha1.Jaeger, metricLine string) v1alpha1.Jaeger {
	// process only lines with collector metrics
	if !strings.HasPrefix(metricLine, "jaeger_collector_") {
		return jaeger
	}

	// at this point, a line look like this:
	// jaeger_collector_spans_dropped_total{host="d096132db661"} 0

	parts := strings.SplitN(metricLine, " ", 2)
	metric := parts[0]

	// metric is like: jaeger_collector_spans_received_total{debug="false",format="jaeger",svc="other-services"}
	if strings.HasPrefix(metric, "jaeger_collector_spans_received_total") {
		jaeger.Status.CollectorSpansReceived = jaeger.Status.CollectorSpansReceived + valueFor(jaeger, parts[1])
	}

	// metric is like: jaeger_collector_spans_dropped_total{host="d096132db661"}
	if strings.HasPrefix(metric, "jaeger_collector_spans_dropped_total") {
		jaeger.Status.CollectorSpansDropped = jaeger.Status.CollectorSpansDropped + valueFor(jaeger, parts[1])
	}

	return jaeger
}

func valueFor(jaeger v1alpha1.Jaeger, v string) int {
	value, err := strconv.Atoi(v)
	if err != nil {
		jaeger.Logger().WithFields(log.Fields{
			"value": v,
		}).WithError(err).Error("failed to parse metric value to int")
		return 0
	}
	return value
}
