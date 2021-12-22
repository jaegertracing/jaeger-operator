#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for streaming-simple test"
cd streaming-simple
export CLUSTER_NAME=my-cluster
export REPLICAS=1
export JAEGER_SERVICE=simple-streaming
export JAEGER_OPERATION=smoketestoperation
export JAEGER_NAME=simple-streaming
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./02-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./04-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./06-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./06-assert.yaml

cd ..

echo "Rendering templates for streaming-with-tls test"
cd streaming-with-tls
export JAEGER_SERVICE=streaming-with-tls
export JAEGER_NAME=tls-streaming
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./02-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./04-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./07-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./07-assert.yaml

cd ..

echo "Rendering templates for streaming-with-autoprovisioning test"
cd streaming-with-autoprovisioning
export CLUSTER_NAME=auto-provisioned
export REPLICAS=3
export JAEGER_SERVICE=streaming-with-autoprovisioning
export JAEGER_NAME=auto-provisioned
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-install.yaml.template -o ./01-install.yaml
$GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./01-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./02-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./04-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./05-smoke-test.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./05-assert.yaml
