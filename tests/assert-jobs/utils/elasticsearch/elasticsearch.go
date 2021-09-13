package elasticsearch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// EsConnection details to the ElasticSearch database
type EsConnection struct {
	Port      string
	URL       string
	Namespace string
}

// PrettyString prints the ES connection details in a nice way
// callback: function to use to print the information
func (connection *EsConnection) PrettyString(callback func(args ...interface{})) {
	callback("ElasticSearch connection details:")
	callback(fmt.Sprintf("\t * Port: %s", connection.Port))
	callback(fmt.Sprintf("\t * URL: %s", connection.URL))
	callback(fmt.Sprintf("\t * Namespace: %s", connection.Namespace))
}

// EsIndex maps indices data from es REST API response
// API endpoint: /_cat/indices?format=json
type EsIndex struct {
	Index        string `json:"index"`
	RealDocCount int
}

// CheckESConnection checs if the connection to ElasticSearch can be done
// es: connection details to the ElasticSearch database
func CheckESConnection(es EsConnection) error {
	_, err := executeEsRequest(es, http.MethodGet, "/")
	if err != nil {
		return fmt.Errorf(fmt.Sprint("There was a problem while connecting to the ES instance: ", err))
	}
	return nil
}

// FormatEsIndices formats the ES Indices information to print it or something
// esIndices: indices to format
// prefix: a prefix for each ES index
// postfix: a postfix for each ES index
func FormatEsIndices(esIndices []EsIndex, prefix, postfix string) string {
	output := ""
	for _, index := range esIndices {
		output = fmt.Sprintf("%s%s%s%s", output, prefix, index.Index, postfix)
	}
	return output
}

// GetEsIndex gets information from an specific ElasticSearch index
// es: connection details to the ElasticSearch database
// indexName: name of the index
func GetEsIndex(es EsConnection, indexName string) (EsIndex, error) {
	index := EsIndex{Index: indexName}
	var err error

	index.RealDocCount, err = getDocCountFromIndex(es, indexName)

	if err != nil {
		return EsIndex{}, err
	}
	return index, nil
}

// GetEsIndices returns the indices from the ElasticSearch node
// es: connection details to the ElasticSearch database
func GetEsIndices(es EsConnection) ([]EsIndex, error) {
	bodyBytes, err := executeEsRequest(es, http.MethodGet, "/_cat/indices?format=json")
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Something failed while quering the ES REST API: %s", err))
	}

	// Convert JSON data to struct format
	esIndices := make([]EsIndex, 0)
	err = json.Unmarshal(bodyBytes, &esIndices)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Something failed while unmarshalling API response: %s", err))
	}

	for i := range esIndices {
		esIndices[i].RealDocCount, err = getDocCountFromIndex(es, esIndices[i].Index)
	}

	return esIndices, nil
}

func getDocCountFromIndex(es EsConnection, indexName string) (int, error) {
	countResponse := struct {
		Count int `json:"count"`
	}{}

	bodyBytes, err := executeEsRequest(es, http.MethodGet, fmt.Sprintf("/%s/_count?format=json", indexName))
	if err != nil {
		return -1, fmt.Errorf(fmt.Sprintf("Something failed while quering the ES REST API: %s", err))
	}

	err = json.Unmarshal(bodyBytes, &countResponse)
	if err != nil {
		return -1, fmt.Errorf(fmt.Sprintf("Something failed while unmarshalling API response: %s", err))
	}
	return countResponse.Count, nil
}

// Executes a REST API ElasticSearch request
// es: connection details to the ElasticSearch database
// httpMethod: HTTP method to use for the query
// api: API endpoint to query
func executeEsRequest(es EsConnection, httpMethod, api string) ([]byte, error) {
	esURL := fmt.Sprintf("%s:%s%s", es.URL, es.Port, api)

	// Create the HTTP client to interact with the API
	transport := &http.Transport{}
	client := http.Client{Transport: transport}

	req, err := http.NewRequest(httpMethod, esURL, nil)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("The HTTP client creation failed: %s", err))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("The HTTP request failed: %s", err))
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
