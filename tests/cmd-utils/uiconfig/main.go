package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
)

const (
	envExpectedContent = "EXPECTED_CONTENT"
	envQueryHostName   = "QUERY_HOSTNAME"
	envAssertPresent   = "ASSERT_PRESENT"
)

func main() {
	viper.AutomaticEnv()

	params := utils.NewParameters()
	params.Parse()

	expectedContent := viper.GetString(envExpectedContent)
	hostName := viper.GetString(envQueryHostName)
	assertPresent := viper.GetBool(envAssertPresent)

	if expectedContent == "" {
		logrus.Fatalln("EXPECTED_CONTENT env variable could not be empty")
	}

	logrus.Info("Querying ", hostName, "...")

	err := utils.TestGetHTTP(hostName, params, func(_ *http.Response, body []byte) (done bool, err error) {
		found := strings.Contains(string(body), expectedContent)

		if assertPresent && found {
			logrus.Infoln("Content found and asserted!")
			return true, nil
		} else if !assertPresent && !found {
			logrus.Infoln("Content not found and asserted it was not found!")
			return true, nil
		}
		logrus.Warningln("Found: ", found, ". Assert: ", assertPresent)
		return false, nil
	})
	if err != nil {
		log.Fatalln(err)
	}

	logrus.Info("Success!")
}
