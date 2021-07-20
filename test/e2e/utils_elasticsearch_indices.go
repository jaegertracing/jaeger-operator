package e2e

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
)

const ElasticSearchIndexDateLayout = "2006-01-02" // date layout in elasticsearch indices, example:

type esIndexData struct {
	IndexName  string    // original index name
	Type       string    // index type. span or service?
	Prefix     string    // prefix of the index
	Date       time.Time // index day/date
	RolloverID string    // rollover string
	DocCount   int       // Document count
}

// Get Jaeger ES indices
// returns in order: serviceIndices, spansIndices
func GetJaegerIndices(namespace string) ([]esIndexData, []esIndexData) {
	esIndices, err := GetEsIndices(namespace)
	require.NoError(t, err)

	servicesIndices := make([]esIndexData, 0)
	spansIndices := make([]esIndexData, 0)

	// parse date, prefix, type from index
	jaegerRe := regexp.MustCompile(`jaeger-(span|service)-`)
	dateRe := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	rolloverRe := regexp.MustCompile(`\d{6}`)

	for _, esIndex := range esIndices {
		if !jaegerRe.MatchString(esIndex.Index) { // assume this index not belongs to Jaeger
			continue
		}

		esData := esIndexData{
			IndexName: esIndex.Index,
		}

		esData.DocCount, err = strconv.Atoi(esIndex.DocsCount)
		require.NoError(t, err)

		dateString := dateRe.FindString(esIndex.Index)
		indexName := strings.Replace(esIndex.Index, dateString, "", 1)

		var indexDate time.Time
		indexDate, err = time.Parse(ElasticSearchIndexDateLayout, dateString)
		if err != nil {
			esData.RolloverID = rolloverRe.FindString(dateString)
			require.NotSame(t, esData.RolloverID, "", "Not date or rollover ID in index")
		}
		esData.Date = indexDate

		// reference
		// https://github.com/jaegertracing/jaeger/blob/6c2be456ca41cdb98ac4b81cb8d9a9a9044463cd/plugin/storage/es/spanstore/reader.go#L40
		if strings.Contains(indexName, "jaeger-span-") {
			esData.Type = "span"
			prefix := strings.Replace(indexName, "jaeger-span-", "", 1)
			if len(prefix) > 0 {
				esData.Prefix = prefix[:len(prefix)-1] // removes "-" at end
			}
			spansIndices = append(spansIndices, esData)
		} else if strings.Contains(indexName, "jaeger-service-") {

			esData.Type = "service"
			prefix := strings.Replace(indexName, "jaeger-service-", "", 1)
			if len(prefix) > 0 {
				esData.Prefix = prefix[:len(prefix)-1] // removes "-" at end
			}
			servicesIndices = append(servicesIndices, esData)
		}
	}

	return servicesIndices, spansIndices
}

func FindIndex(indices []esIndexData, name string) (esIndexData, error) {
	for _, esIndex := range indices {
		if esIndex.IndexName == name {
			return esIndex, nil
		}
	}
	return esIndexData{}, errors.New("Index not found")
}
