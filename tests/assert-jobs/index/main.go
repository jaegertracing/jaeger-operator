package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils/elasticsearch"
)

const (
	flagEsNamespace        = "es-namespace"
	flagEsPort             = "es-port"
	flagEsURL              = "es-url"
	flagPattern            = "pattern"
	flagName               = "name"
	flagExist              = "assert-exist"
	flagAssertCountIndices = "assert-count-indices"
	flagAssertCountDocs    = "assert-count-docs"
	flagJaegerService      = "jaeger-service"
	flagCertificatePath    = "certificate-path"
	flagVerbose            = "verbose"
)

func filterIndices(indices *[]elasticsearch.EsIndex, pattern string) ([]elasticsearch.EsIndex, error) {
	regexPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("there was a problem with the pattern: %w", err)
	}

	var matchingIndices []elasticsearch.EsIndex

	for _, index := range *indices {
		if regexPattern.MatchString(index.Index) {
			logrus.Debugf("Index '%s' matched", index.Index)
			matchingIndices = append(matchingIndices, index)
		}
	}

	logrus.Debugf("%d indices matches the pattern '%s'", len(matchingIndices), pattern)

	return matchingIndices, nil
}

// Init the CMD and return error if something didn't go properly
func initCmd() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(flagEsNamespace, "default")
	flag.String(flagEsNamespace, "", "ElasticSearch namespace to use")

	viper.SetDefault(flagEsPort, "9200")
	flag.String(flagEsPort, "", "ElasticSearch port")

	viper.SetDefault(flagEsURL, "http://localhost")
	flag.String(flagEsURL, "", "ElasticSearch URL")

	viper.SetDefault(flagVerbose, false)
	flag.Bool(flagVerbose, false, "Enable verbosity")

	viper.SetDefault(flagExist, false)
	flag.Bool(flagExist, false, "Assert the pattern matches something")

	viper.SetDefault(flagPattern, "")
	flag.String(flagPattern, "", "Pattern to use to match indices")

	viper.SetDefault(flagName, "")
	flag.String(flagName, "", "Name of the desired index (needed for aliases)")

	viper.SetDefault(flagJaegerService, "")
	flag.String(flagJaegerService, "", "Name of the Jaeger Service")

	viper.SetDefault(flagAssertCountIndices, "-1")
	flag.Int(flagAssertCountIndices, -1, "Assert the number of matched indices")

	viper.SetDefault(flagAssertCountDocs, "-1")
	flag.Int(flagAssertCountDocs, -1, "Assert the number of documents")

	viper.SetDefault(flagCertificatePath, "")
	flag.String(flagCertificatePath, "", "Path to the secret to use during the API calls")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}
	params := utils.NewParameters()
	params.Parse()

	if viper.GetString(flagName) != "" && viper.GetString(flagPattern) != "" {
		return fmt.Errorf("--%s and --%s provided. Provide just one", flagName, flagPattern)
	} else if viper.GetString(flagName) == "" && viper.GetString(flagPattern) == "" {
		return fmt.Errorf("--%s nor --%s provided. Provide one at least", flagName, flagPattern)
	} else if viper.GetBool(flagAssertCountDocs) && viper.GetString(flagJaegerService) == "" {
		return fmt.Errorf("--%s provided. Provide --%s", flagAssertCountDocs, flagJaegerService)
	}

	return nil
}

func main() {
	err := initCmd()
	if err != nil {
		logrus.Fatalln(err)
	}

	if viper.GetBool(flagVerbose) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	connection := elasticsearch.EsConnection{
		Port:        viper.GetString(flagEsPort),
		Namespace:   viper.GetString(flagEsNamespace),
		URL:         viper.GetString(flagEsURL),
		RootCAs:     nil,
		Certificate: tls.Certificate{},
	}
	connection.PrettyString(logrus.Debug)

	if viper.GetString(flagCertificatePath) != "" {
		err = connection.LoadCertificate(viper.GetString(flagCertificatePath))
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	err = elasticsearch.CheckESConnection(connection)
	if err != nil {
		logrus.Fatalln(err)
	}

	var matchingIndices []elasticsearch.EsIndex
	if viper.GetString(flagPattern) != "" {
		indices, err := elasticsearch.GetEsIndices(connection)
		if err != nil {
			logrus.Fatalln("There was an error while getting the ES indices: ", err)
		}

		matchingIndices, err = filterIndices(&indices, viper.GetString(flagPattern))
		if err != nil {
			logrus.Fatalln(err)
		}
	} else {
		index := elasticsearch.GetEsIndex(connection, viper.GetString(flagName))
		matchingIndices = []elasticsearch.EsIndex{index}
	}

	if viper.GetBool(flagExist) {
		if len(matchingIndices) == 0 {
			logrus.Fatalln("No indices match the pattern")
		}
	}

	if viper.GetString(flagName) != "" && viper.GetString(flagAssertCountIndices) != "" {
		logrus.Warnln("Ignoring parameter", flagAssertCountIndices, "because we are checking the info for one index")
	} else if viper.GetString(flagPattern) != "" && viper.GetInt(flagAssertCountIndices) > -1 {
		if len(matchingIndices) != viper.GetInt(flagAssertCountIndices) {
			logrus.Fatalln(len(matchingIndices), "indices found.", viper.GetInt(flagAssertCountIndices), "expected")
		}
	}

	if viper.GetInt(flagAssertCountDocs) > -1 {
		foundDocs := 0
		jaegerServiceName := viper.GetString(flagJaegerService)
		for _, index := range matchingIndices {
			spans, err := index.GetServiceIndexSpans(jaegerServiceName)
			if err != nil {
				logrus.Errorln("Something failed while getting the index spans:", err)
			}
			foundDocs += len(spans)
		}
		logrus.Debug(foundDocs, " in ", len(matchingIndices), " indices")

		if foundDocs != viper.GetInt(flagAssertCountDocs) {
			logrus.Fatalln(foundDocs, "docs found.", viper.GetInt(flagAssertCountDocs), "expected")
		}
	}
}
