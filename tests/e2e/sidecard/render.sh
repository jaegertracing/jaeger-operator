#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for sidecar-agent test"
cd sidecar-agent
export JAEGER_NAME=agent-as-sidecar
export JAEGER_SERVICE=order
JOB_NUMBER=1 $GOMPLATE -f $TEMPLATES_DIR/find-service.yaml.template -o ./02-find-service.yaml
JOB_NUMBER=1 $GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./02-assert.yaml
export JAEGER_NAME=agent-as-sidecar2
JOB_NUMBER=2 $GOMPLATE -f $TEMPLATES_DIR/find-service.yaml.template -o ./05-find-service-other-instance.yaml
JOB_NUMBER=2 $GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./05-assert.yaml

cd ..

echo "Rendering templates for sidecar-namespace test"
cd sidecar-namespace
export JAEGER_NAME=agent-as-sidecar
export JAEGER_SERVICE=order
JOB_NUMBER=1 $GOMPLATE -f $TEMPLATES_DIR/find-service.yaml.template -o ./02-find-service.yaml
JOB_NUMBER=1 $GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./02-assert.yaml
export JAEGER_NAME=agent-as-sidecar2
JOB_NUMBER=2 $GOMPLATE -f $TEMPLATES_DIR/find-service.yaml.template -o ./05-find-service-other-instance.yaml
JOB_NUMBER=2 $GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./05-assert.yaml
