package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

//TestGetHTTP polls an endpoint and test the response
func TestGetHTTP(url string, params *TestParams, testFn func(response *http.Response, body []byte) (done bool, err error)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return err
	}

	client := http.Client{Timeout: 30 * time.Second}

	logrus.Info("Polling to %s", url)

	return wait.Poll(params.RetryInterval, params.Timeout, func() (done bool, err error) {
		logrus.Info("Doing request..")
		res, err := client.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			return false, nil
		}

		if err != nil {
			return false, err
		}

		if res.StatusCode != http.StatusOK {
			return false, fmt.Errorf("unexpected status code %d", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return false, err
		}

		if len(body) == 0 {
			return false, fmt.Errorf("empty body")
		}

		return testFn(res, body)
	})
}
