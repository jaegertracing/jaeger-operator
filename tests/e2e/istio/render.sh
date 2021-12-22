#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for istio test"
cd istio
export JAEGER_NAME=simplest
export JAEGER_SERVICE=order
cat $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./livelinessprobe.template > ./03-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/find-service.yaml.template -o ./04-smoke.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./04-assert.yaml
