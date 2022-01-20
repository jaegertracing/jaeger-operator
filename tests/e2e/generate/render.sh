#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "generate"
# JAEGER_VERSION environment variable is set before this script is called
export JAEGER_NAME=my-jaeger

$GOMPLATE -f ./jaeger-template.yaml.template -o ./jaeger-deployment.yaml

render_smoke_test "$JAEGER_NAME" "allInOne" "01"
