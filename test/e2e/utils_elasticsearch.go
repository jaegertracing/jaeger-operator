package e2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
)

// EsIndex struct to map indices data from es rest api response
// es api: /_cat/indices?format=json
type EsIndex struct {
	UUID        string `json:"uuid"`
	Status      string `json:"status"`
	Index       string `json:"index"`
	Health      string `json:"health"`
	DocsCount   string `json:"docs.count"`
	DocsDeleted string `json:"docs.deleted"`
	StoreSize   string `json:"store.size"`
}

// GetEsIndices return indices from es node
func GetEsIndices(esNamespace string) ([]EsIndex, error) {
	bodyBytes, err := ExecuteEsRequest(esNamespace, http.MethodGet, "/_cat/indices?format=json")
	require.NoError(t, err)

	// convert json data to struct format
	esIndices := make([]EsIndex, 0)
	err = json.Unmarshal(bodyBytes, &esIndices)
	require.NoError(t, err)

	return esIndices, nil
}

// DeleteEsIndices deletes all the indices on es node
func DeleteEsIndices(esNamespace string) {
	logrus.Info("deleting all es node indices")
	_, err := ExecuteEsRequest(esNamespace, http.MethodDelete, "/_all?format=json")
	require.NoError(t, err)
}

// ExecuteEsRequest executes rest api request on es node
func ExecuteEsRequest(esNamespace, httpMethod, api string) ([]byte, error) {
	// enable port forward
	fwdPortES, closeChanES, esPort := CreateEsPortForward(esNamespace)
	defer fwdPortES.Close()
	defer close(closeChanES)

	// update es node url
	urlScheme := "http"
	if skipESExternal {
		urlScheme = "https"
	}
	esURL := fmt.Sprintf("%s://localhost:%s%s", urlScheme, esPort, api)

	// create rest client to access es node rest API
	transport := &http.Transport{}
	client := http.Client{Transport: transport}

	// update certificates, if the es node provided by jaeger-operator
	if skipESExternal {
		esSecret, err := fw.KubeClient.CoreV1().Secrets(namespace).Get(context.Background(), "elasticsearch", metav1.GetOptions{})
		require.NoError(t, err)
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(esSecret.Data["admin-ca"])

		clientCert, err := tls.X509KeyPair(esSecret.Data["admin-cert"], esSecret.Data["admin-key"])
		require.NoError(t, err)

		transport.TLSClientConfig = &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{clientCert},
		}
	}

	req, err := http.NewRequest(httpMethod, esURL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.EqualValues(t, 200, resp.StatusCode)

	return ioutil.ReadAll(resp.Body)
}

// CreateEsPortForward creates local port forwarding
func CreateEsPortForward(esNamespace string) (portForwES *portforward.PortForwarder, closeChanES chan struct{}, esPort string) {
	portForwES, closeChanES = CreatePortForward(esNamespace, string(v1.JaegerESStorage), string(v1.JaegerESStorage), []string{"0:9200"}, fw.KubeConfig)
	forwardedPorts, err := portForwES.GetPorts()
	require.NoError(t, err)
	return portForwES, closeChanES, strconv.Itoa(int(forwardedPorts[0].Local))
}
