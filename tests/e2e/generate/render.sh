#!/bin/bash

source $(dirname "$0")/../render-utils.sh

###############################################################################
# TEST NAME: generate
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    skip_test "generate" "This test was skipped until https://github.com/jaegertracing/jaeger-operator/issues/2145 is fixed"
else
    start_test "generate"
    # JAEGER_VERSION environment variable is set before this script is called
    jaeger_name="my-jaeger"

    $GOMPLATE -f ./jaeger-template.yaml.template -o ./jaeger-deployment.yaml

    render_smoke_test "$jaeger_name" "false" "01"
fi