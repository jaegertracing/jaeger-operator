#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for es-index-cleaner test"
cd es-index-cleaner
export JAEGER_NAME=test-es-index-cleaner-with-prefix
export PREFIX=my-prefix
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-install.yaml.template -o ./jaeger-deployment
$GOMPLATE -f ./es-index.template -o ./es-index
cat ./jaeger-deployment ./es-index >> ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./01-assert.yaml
sed "s~enabled: false~enabled: true~gi" ./01-install.yaml > ./03-install.yaml
$GOMPLATE -f ./01-install.yaml -o ./05-install.yaml
$GOMPLATE -f ./es-index.template -o ./es-index2
cat ./jaeger-deployment ./es-index2 >> ./07-install.yaml
sed "s~enabled: false~enabled: true~gi" ./07-install.yaml > ./09-install.yaml
$GOMPLATE -f ./04-wait-es-index-cleaner.yaml -o ./11-wait-es-index-cleaner.yaml
$GOMPLATE -f ./05-install.yaml -o ./12-install.yaml

cd ..

echo "Rendering templates for es-simple-prod test"
cd es-simple-prod
export JAEGER_NAME=simple-prod
export JAEGER_SERVICE=simple-prod
export JAEGER_OPERATION=smoketestoperation
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-install.yaml.template -o ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./02-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./02-assert.yaml

cd ..

echo "Rendering templates for es-spark-dependencies test"
cd es-spark-dependencies
$GOMPLATE -f  $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./00-install.yaml
$GOMPLATE -f  $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./00-assert.yaml
