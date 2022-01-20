package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
)

type services struct {
	Data   []string    `json:"data"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Errors interface{} `json:"errors"`
}

const (
	flagIngressHost = "query-host"
	flagServiceName = "service-name"
	flagVerbose     = "verbose"
)

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(flagIngressHost, "http://localhost")
	flag.String(flagIngressHost, "", "Query service hostname")
	flag.String(flagServiceName, "", "Service name expected")
	viper.SetDefault(flagVerbose, false)
	flag.Bool(flagVerbose, false, "Enable verbosity")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	params := utils.NewParameters()
	params.Parse()

	host := viper.GetString(flagIngressHost)
	serviceName := viper.GetString(flagServiceName)

	if viper.GetBool(flagVerbose) == true {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetOutput(os.Stdout)

	url := fmt.Sprintf("%s/api/services", host)

	err = utils.TestGetHTTP(url, params, func(response *http.Response, body []byte) (done bool, err error) {
		resp := &services{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			logrus.Warn("There was an error unmarshalling the response:", err.Error())
			return false, err
		}
		for _, v := range resp.Data {
			logrus.Debug("Found service '", v, "'")
			logrus.Debug(serviceName)

			if v == serviceName {
				logrus.Info("The service was found!!")
				return true, nil
			}
		}

		logrus.Debug(resp.Data)
		return false, nil
	})

	if err != nil {
		logrus.Error("Error querying the Jaeger instance: ", err)
		os.Exit(1)
	}
	logrus.Info("Successfully terminates")
}
