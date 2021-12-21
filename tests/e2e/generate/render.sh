#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for generate test"
cd generate
# JAEGER_VERSION environment variable is set before this script is called
export JAEGER_NAME=my-jaeger
export JAEGER_SERVICE=test-service
export JAEGER_OPERATION=smoketestoperation
$GOMPLATE -f ./jaeger-template.yaml.template -o ./jaeger-deployment.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml
