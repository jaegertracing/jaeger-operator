package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

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
	flagIngressHost = "ingress-host"
	flagServiceName = "service-name"
)

func main() {
	viper.AutomaticEnv()

	flag.String(flagServiceName, "", "Service name expected")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(err)
	}

	params := utils.NewParameters()
	params.Parse()

	viper.SetDefault(flagIngressHost, "localhost")
	host := viper.GetString(flagIngressHost)
	serviceName := viper.GetString(flagServiceName)

	url := fmt.Sprintf("http://%s/api/services", host)

	err = utils.TestGetHTTP(url, params, func(response *http.Response, body []byte) (done bool, err error) {
		resp := &services{}
		err = json.Unmarshal(body, &resp)
		for _, v := range resp.Data {
			if v == serviceName {
				return true, nil
			}
		}
		return false, nil
	})

	if err != nil {
		logrus.Error("Error trying to parse response: %v", err)
		os.Exit(1)
	}
}
