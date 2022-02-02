#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "streaming-simple"
render_install_elasticsearch "00"
render_install_kafka "my-cluster" "1" "01"
render_smoke_test "simple-streaming" "production" "05"


start_test "streaming-with-tls"
render_install_kafka "my-cluster" "1" "00"
render_install_elasticsearch "01"
render_smoke_test "tls-streaming" "production" "06"

start_test "streaming-with-autoprovisioning"
export CLUSTER_NAME="auto-provisioned"
export REPLICAS=3
jaeger_name="auto-provisioned"
render_install_elasticsearch "01"
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./04-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./05-assert.yaml
render_smoke_test "$jaeger_name" "production" "06"
