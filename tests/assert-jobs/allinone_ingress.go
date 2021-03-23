package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
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

const ingressHostKey = "ingress-host"

func main() {

	params := utils.NewParameters()
	params.Parse()
	viper.SetDefault(ingressHostKey, "localhost")
	host := viper.GetString(ingressHostKey)
	url := fmt.Sprintf("http://%s/api/services", host)

	// Hit this url once to make Jaeger itself create a trace, then it will show up in services
	httpClient := http.Client{Timeout: 2 * time.Second}
	_, err := httpClient.Get(url)

	if err != nil {
		log.Print(err)
	}

	err = utils.TestGetHTTP(url, params, func(response *http.Response, body []byte) (done bool, err error) {
		resp := &services{}
		err = json.Unmarshal(body, &resp)
		for _, v := range resp.Data {
			if v == "jaeger-query" {
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
