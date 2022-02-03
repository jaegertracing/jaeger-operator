#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "streaming-simple"
render_install_elasticsearch "00"
render_install_kafka "my-cluster" "1" "01"
JAEGER_NAME="simple-streaming" $GOMPLATE -f $TEMPLATES_DIR/streaming-jaeger-assert.yaml.template -o ./04-assert.yaml
render_smoke_test "simple-streaming" "production" "05"


start_test "streaming-with-tls"
render_install_kafka "my-cluster" "1" "00"
render_install_elasticsearch "01"
render_smoke_test "tls-streaming" "allInOne" "05"

start_test "streaming-with-autoprovisioning"
export CLUSTER_NAME="auto-provisioned"
jaeger_name="auto-provisioned"

if [ $IS_OPENSHIFT = true ]; then
    # Remove the installation of the operator
    rm ./00-install.yaml ./00-assert.yaml
    REPLICAS=1
else
    REPLICAS=3
fi
export REPLICAS

render_install_elasticsearch "01"
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./04-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./05-assert.yaml
render_smoke_test "$jaeger_name" "allInOne" "06"
