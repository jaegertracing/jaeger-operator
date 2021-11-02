package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
)

const (
	envQueryBasePath = "QUERY_BASE_PATH"
	envTracingIDKey  = "TRACING_ID"
	envQueryHostName = "QUERY_HOSTNAME"
)

func main() {
	viper.AutomaticEnv()

	params := utils.NewParameters()
	params.Parse()

	basePath := viper.GetString(envQueryBasePath)
	trackingID := viper.GetString(envTracingIDKey)
	hostName := viper.GetString(envQueryHostName)

	searchURL := fmt.Sprintf("http://%s:16686/%s/search", hostName, basePath)

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
