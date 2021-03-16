package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
)

func main() {
	params := utils.NewParameters()
	params.Parse()

	basePath := viper.GetString("QUERY_BASE_PATH")
	trackingID := viper.GetString("TRACKING_ID")

	serviceName := service.GetNameForQueryService(v1.NewJaeger(types.NamespacedName{Name: params.JaegerName}))
	searchURL := fmt.Sprintf("http://%s:16686/%s/search", serviceName, basePath)

	err := utils.TestGetHTTP(searchURL, params, func(_ *http.Response, body []byte) (done bool, err error) {
		if !strings.Contains(string(body), trackingID) {
			return false, fmt.Errorf("body does not include tracking id: %s", trackingID)
		}

		return true, nil
	})
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	logrus.Info("Success")
}
