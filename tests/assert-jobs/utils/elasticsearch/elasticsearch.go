package elasticsearch

import (
	"bytes"
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

// EsSpan maps spans data from ES REST API response
// API endpoint: /<index>/_search?format=json
type EsSpan struct {
	ID            string
	ServiceName   string
	OperationName string
}

// PrettyString prints the ES connection details in a nice way
// callback: function to use to print the information
func (connection *EsConnection) PrettyString(callback func(args ...interface{})) {
	callback("ElasticSearch connection details:")
	callback(fmt.Sprintf("\t * Port: %s", connection.Port))
	callback(fmt.Sprintf("\t * URL: %s", connection.URL))
	callback(fmt.Sprintf("\t * Namespace: %s", connection.Namespace))
}

// EsIndex maps indices data from ES REST API response
// API endpoint: /_cat/indices?format=json
type EsIndex struct {
	Index string `json:"index"`
	es    EsConnection
}

// GetServiceIndexSpans gets the spans associated to one index and one service
// serviceName: name of the Jaeger service
func (index *EsIndex) GetServiceIndexSpans(serviceName string) ([]EsSpan, error) {
	spans, err := index.GetIndexSpans()
	if err != nil {
		return []EsSpan{}, err
	}

	filteredSpans := []EsSpan{}

	for _, span := range spans {
		if span.ServiceName == serviceName {
			filteredSpans = append(filteredSpans, span)
		}
	}
	return filteredSpans, nil
}

// GetIndexSpans gets the spans associated to one index
func (index *EsIndex) GetIndexSpans() ([]EsSpan, error) {
	searchResponse := struct {
		Hits struct {
			Hits []struct {
				ID     string `json:"_id"`
				Source struct {
					Process struct {
						ServiceName string `json:"serviceName"`
					} `json:"process"`
					OperationName string `json:"operationName"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}{}

	body := struct {
		Query struct {
			QueryString struct {
				Query string `json:"query"`
			} `json:"query_string"`
		} `json:"query"`
		Size int `json:"size"`
		From int `json:"from"`
	}{}
	body.From = 0
	body.Size = 10000
	body.Query.QueryString.Query = "*"

	bodyReq, _ := json.Marshal(body)

	bodyBytes, err := executeEsRequest(index.es, http.MethodPost, fmt.Sprintf("/%s/_search?format=json", index.Index), bodyReq)
	if err != nil {
		return []EsSpan{}, fmt.Errorf(fmt.Sprintf("Something failed while quering the ES REST API: %s", err))
	}

	err = json.Unmarshal(bodyBytes, &searchResponse)
	if err != nil {
		return []EsSpan{}, fmt.Errorf(fmt.Sprintf("Something failed while unmarshalling API response: %s", err))
	}

	spans := []EsSpan{}
	for _, jsonSpan := range searchResponse.Hits.Hits {
		span := EsSpan{ID: jsonSpan.ID, ServiceName: jsonSpan.Source.Process.ServiceName, OperationName: jsonSpan.Source.OperationName}
		spans = append(spans, span)
	}
	return spans, nil
}

// CheckESConnection checs if the connection to ElasticSearch can be done
// es: connection details to the ElasticSearch database
func CheckESConnection(es EsConnection) error {
	_, err := executeEsRequest(es, http.MethodGet, "/", nil)
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
func GetEsIndex(es EsConnection, indexName string) EsIndex {
	return EsIndex{indexName, es}
}

// GetEsIndices returns the indices from the ElasticSearch node
// es: connection details to the ElasticSearch database
func GetEsIndices(es EsConnection) ([]EsIndex, error) {
	bodyBytes, err := executeEsRequest(es, http.MethodGet, "/_cat/indices?format=json", nil)
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
		esIndices[i].es = es
	}

	return esIndices, nil
}

// Executes a REST API ElasticSearch request
// es: connection details to the ElasticSearch database
// httpMethod: HTTP method to use for the query
// api: API endpoint to query
func executeEsRequest(es EsConnection, httpMethod, api string, body []byte) ([]byte, error) {
	esURL := fmt.Sprintf("%s:%s%s", es.URL, es.Port, api)

	// Create the HTTP client to interact with the API
	transport := &http.Transport{}
	client := http.Client{Transport: transport}

	var bodyReq []byte
	var err error

	if body == nil {
		bodyReq = nil
	} else {
		bodyReq = body
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("Something failed while marshalling the body: %s", err))
		}
	}

	req, err := http.NewRequest(httpMethod, esURL, bytes.NewBuffer(bodyReq))
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("The HTTP client creation failed: %s", err))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("The HTTP request failed: %s", err))
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
