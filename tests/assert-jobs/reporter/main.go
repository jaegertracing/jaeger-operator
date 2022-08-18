package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
)

const (
	flagJaegerServiceName   = "jaeger-service-name"
	flagJaegerOperationName = "operation-name"
	flagDays                = "days"
	flagVerbose             = "verbose"
	flagServices            = "services"
	envVarJaegerEndpoint    = "jaeger_endpoint"
	enVarJaegerQuery        = "jaeger_query"
)

// Init the Jaeger tracer. Returns the tracer and the closer.
// serviceName: name of the service to report spans
func initTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg, err := config.FromEnv()
	if serviceName != "" {
		cfg.ServiceName = serviceName
	}

	cfg.Reporter.LogSpans = viper.GetBool(flagVerbose)
	cfg.Sampler = &config.SamplerConfig{
		Type:  "const",
		Param: 1,
	}
	if err != nil {
		panic(err)
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

// Assert the span was reported properly
// spanDate: start date of the reported span
// serviceName: name of the span service
func assertSpanWasCreated(spanDate time.Time, serviceName string) bool {
	startQueryTime := spanDate.Add(time.Minute * -2)
	finishQueryTime := spanDate.Add(time.Minute * 2)

	jaegerCollectorEndpoint := viper.GetString(enVarJaegerQuery)

	url := fmt.Sprintf(
		"%s?lookback=custom&service=%s&limit=200&start=%d&end=%d",
		jaegerCollectorEndpoint,
		serviceName,
		startQueryTime.UnixNano()/1000,
		finishQueryTime.UnixNano()/1000,
	)
	params := utils.TestParams{}
	params.Parse()
	params.Timeout = time.Minute
	params.RetryInterval = time.Second * 5

	err := utils.TestGetHTTP(url, &params, func(response *http.Response, body []byte) (done bool, err error) {
		resp := struct {
			Data []struct {
				Spans []struct {
					StartTime int64 `json:"startTime"`
				} `json:"spans"`
			} `json:"data"`
		}{}

		err = json.Unmarshal(body, &resp)
		if err != nil {
			return false, err
		}

		for _, reportedTrace := range resp.Data {
			for _, reportedSpan := range reportedTrace.Spans {
				if reportedSpan.StartTime == spanDate.UnixNano()/1000 {
					return true, nil
				}
			}
		}

		return false, nil
	})
	if err == nil {
		logrus.Info("Span asserted properly")
		return true
	}
	logrus.Error("There was a problem reporting the information: ", err)
	return false
}

// Generate spans for the given service
// serviceName: name of the service to generate spans
// operationName: name of the operation for the spans
// days: number of days to generate spans
func generateSpansHistoryService(serviceName, operationName string, days int) {
	if days < 1 {
		logrus.Warn("days parameter for generateSpansHistory is less than 1. Doing nothing")
		return
	}

	logrus.Info("Generating spans for the last ", days, " days for service ", serviceName)

	currentDate := time.Now()
	tracer, closer := initTracer(serviceName)
	defer closer.Close()

	generatedSpans := 0

	for day := 0; day < days; day++ {
		spanDate := currentDate.AddDate(0, 0, -1*day)
		spanOperationName := fmt.Sprintf("%s-%d", operationName, day)

		generateSpan(spanDate, spanOperationName, &tracer)

		jaegerQueryEndpoint := viper.GetString(enVarJaegerQuery)
		if jaegerQueryEndpoint != "" {
			for !assertSpanWasCreated(spanDate, serviceName) {
				generateSpan(spanDate, spanOperationName, &tracer)
			}
			generatedSpans++
			logrus.Info(generatedSpans, " spans reported properly")
		}
	}
}

func generateSpan(spanDate time.Time, operationName string, tracer *opentracing.Tracer) {
	stringDate := spanDate.Format("2 Jan 2006 15:04:05")
	span := (*tracer).StartSpan(operationName, opentracing.StartTime(spanDate))
	span.SetTag("string-date", stringDate)
	span.FinishWithOptions(opentracing.FinishOptions{FinishTime: spanDate.Add(time.Hour * 2)})
}

// Generate spans for multiple services
// serviceName: prefix name name of the services to generate spans
// operationName: name of the operation for the spans
// days: number of days to generate spans
// services: number of services to generate
func generateSpansHistory(serviceName, operationName string, days, services int) {
	for service := 0; service < services; service++ {
		reportedServiceName := serviceName
		if services > 1 {
			reportedServiceName = fmt.Sprintf("%s-%d", serviceName, service)
		}
		generateSpansHistoryService(reportedServiceName, operationName, days)
	}
}

// Init the CMD and return error if something didn't go properly
func initCmd() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(flagJaegerServiceName, "jaeger-service")
	flag.String(flagJaegerServiceName, "", "Jaeger service name")

	viper.SetDefault(flagDays, 1)
	flag.Int(flagDays, 1, "History days")

	viper.SetDefault(flagServices, 1)
	flag.Int(flagServices, 1, "Number of services")

	viper.SetDefault(flagVerbose, false)
	flag.Bool(flagVerbose, false, "Enable verbosity")

	viper.SetDefault(flagJaegerOperationName, "jaeger-operation")
	flag.String(flagJaegerOperationName, "", "Jaeger operation name")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	return err
}

func main() {
	err := initCmd()
	if err != nil {
		logrus.Fatal(err)
	}

	if viper.GetBool(flagVerbose) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	jaegerEndpoint := viper.GetString(envVarJaegerEndpoint)
	if jaegerEndpoint == "" {
		logrus.Fatal("Please, specify a Jaeger Collector endpoint")
	}

	// Sometimes, Kubernetes reports the Jaeger service is there but there is
	// an interval where the service is up but the REST API is not operative yet
	err = utils.WaitUntilRestAPIAvailable(jaegerEndpoint)
	if err != nil {
		logrus.Fatalln(err)
	}

	generateSpansHistory(viper.GetString(flagJaegerServiceName), viper.GetString(flagJaegerOperationName), viper.GetInt(flagDays), viper.GetInt(flagServices))

	// After reporting the spans, we wait some seconds to ensure the spans were reported and
	// stored in the final storage (ES or other)
	time.Sleep(time.Second * 10)
}
