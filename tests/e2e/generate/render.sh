#!/bin/bash

source $(dirname "$0")/../render-utils.sh

is_secured="false"
if [ $IS_OPENSHIFT = true ]; then
    is_secured="true"
fi

###############################################################################
# TEST NAME: generate
###############################################################################
start_test "generate"
# JAEGER_VERSION environment variable is set before this script is called
jaeger_name="my-jaeger"

$GOMPLATE -f ./jaeger-template.yaml.template -o ./jaeger-deployment.yaml

render_smoke_test "$jaeger_name" "$is_secured" "01"
