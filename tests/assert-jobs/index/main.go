package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils"
	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils/elasticsearch"
	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils/logger"
)

var log logrus.Logger

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
	flagVerbose            = "verbose"
)

func filterIndices(indices *[]elasticsearch.EsIndex, pattern string) ([]elasticsearch.EsIndex, error) {
	regexPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("There was a problem with the pattern: %s", err))
	}

	var matchingIndices []elasticsearch.EsIndex

	for _, index := range *indices {
		if regexPattern.MatchString(index.Index) {
			log.Debugf("Index '%s' matched", index.Index)
			matchingIndices = append(matchingIndices, index)
		}
	}

	log.Debugf("%d indices matches the pattern '%s'", len(matchingIndices), pattern)

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

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}
	params := utils.NewParameters()
	params.Parse()

	if viper.GetString(flagName) != "" && viper.GetString(flagPattern) != "" {
		return fmt.Errorf(fmt.Sprintf("--%s and --%s provided. Provide just one", flagName, flagPattern))
	} else if viper.GetString(flagName) == "" && viper.GetString(flagPattern) == "" {
		return fmt.Errorf(fmt.Sprintf("--%s nor --%s provided. Provide one at least", flagName, flagPattern))
	} else if viper.GetBool(flagAssertCountDocs) && viper.GetString(flagJaegerService) == "" {
		return fmt.Errorf(fmt.Sprintf("--%s provided. Provide --%s", flagAssertCountDocs, flagJaegerService))
	}

	return nil
}

func main() {
	err := initCmd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log = *logger.InitLog(viper.GetBool(flagVerbose))

	connection := elasticsearch.EsConnection{
		Port:      viper.GetString(flagEsPort),
		Namespace: viper.GetString(flagEsNamespace),
		URL:       viper.GetString(flagEsURL),
	}
	connection.PrettyString(log.Debug)

	err = elasticsearch.CheckESConnection(connection)
	if err != nil {
		log.Fatalln(err)
		log.Exit(1)
	}

	var matchingIndices []elasticsearch.EsIndex
	if viper.GetString(flagPattern) != "" {
		indices, err := elasticsearch.GetEsIndices(connection)
		if err != nil {
			log.Fatalln("There was an error while getting the ES indices: ", err)
			log.Exit(1)
		}

		matchingIndices, err = filterIndices(&indices, viper.GetString(flagPattern))
		if err != nil {
			log.Fatalln(err)
			os.Exit(1)
		}
	} else {
		index := elasticsearch.GetEsIndex(connection, viper.GetString(flagName))
		matchingIndices = []elasticsearch.EsIndex{index}
	}

	if viper.GetBool(flagExist) {
		if len(matchingIndices) == 0 {
			log.Fatalln("No indices match the pattern")
			os.Exit(1)
		}
	}

	if viper.GetString(flagName) != "" && viper.GetString(flagAssertCountIndices) != "" {
		log.Warnln("Ignoring parameter", flagAssertCountIndices, "because we are checking the info for one index")
	} else if viper.GetString(flagPattern) != "" && viper.GetInt(flagAssertCountIndices) > -1 {
		if len(matchingIndices) != viper.GetInt(flagAssertCountIndices) {
			log.Fatalln(len(matchingIndices), "indices found.", viper.GetInt(flagAssertCountIndices), "expected")
			os.Exit(1)
		}
	}

	if viper.GetInt(flagAssertCountDocs) > -1 {
		foundDocs := 0
		jaegerServiceName := viper.GetString(flagJaegerService)
		for _, index := range matchingIndices {
			spans, err := index.GetServiceIndexSpans(jaegerServiceName)
			if err != nil {
				log.Errorln("Something failed while getting the index spans:", err)
			}
			foundDocs += len(spans)
		}
		log.Debug(foundDocs, " in ", len(matchingIndices), " indices")

		if foundDocs != viper.GetInt(flagAssertCountDocs) {
			log.Fatalln(foundDocs, "docs found.", viper.GetInt(flagAssertCountDocs), "expected")
			os.Exit(1)
		}
	}

}
