package utils

import (
	"crypto/tls"
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

	if params.Secret == "" {
		logrus.Info("No secret provided for the Authorization header")
	} else {
		// This is needed to query the endpoints when using the OAuth Proxy
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", params.Secret))
		logrus.Info("Secret provided for the Authorization header")
	}

	client := http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	maxRetries := 20
	retries := 0
	failed := false

	logrus.Info("Polling to ", url)

	return wait.Poll(params.RetryInterval, params.Timeout, func() (done bool, err error) {
		logrus.Info("Doing request number ", retries)

		res, err := client.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			failed = true
			logrus.Warn("Timeout exceeded!")
		} else if res != nil && res.StatusCode != http.StatusOK {
			err = fmt.Errorf("Unexpected status code %d", res.StatusCode)
			failed = true
			logrus.Warn("Status code: ", res.StatusCode)
		} else if err != nil {
			failed = true
			logrus.Warn("Something failed during doing the request: ", err.Error())
		}

		if failed {
			failed = false
			retries++
			if retries > maxRetries {
				return false, err
			}
			return false, nil
		}

		body, err := ioutil.ReadAll(res.Body)
		if len(body) == 0 {
			failed = true
			err = fmt.Errorf("Empty body response")
			logrus.Warn("Empty body response")
		} else if err != nil {
			failed = true
			logrus.Warn("Something failed reading the response: ", err.Error())
		}

		if failed {
			failed = false
			retries++
			if retries > maxRetries {
				return false, err
			}
			return false, nil
		}

		ok, err := testFn(res, body)
		if ok {
			return true, nil
		}
		retries++

		if retries > maxRetries {
			logrus.Warn("Something failed while executing the test function: ", err.Error())
			return false, err
		}

		if err == nil {
			logrus.Warn("The condition of the test function was not accomplished")
		} else {
			logrus.Warn("There test function returned an error:", err.Error())
		}

		return false, nil

	})
}
