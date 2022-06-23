#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "streaming-simple"
render_install_elasticsearch "00"
render_install_kafka "my-cluster" "1" "01"
JAEGER_NAME="simple-streaming" $GOMPLATE -f $TEMPLATES_DIR/streaming-jaeger-assert.yaml.template -o ./04-assert.yaml
render_smoke_test "simple-streaming" "production" "05"


start_test "streaming-with-tls"
render_install_kafka "my-cluster" "1" "00"
render_install_elasticsearch "03"
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
render_smoke_test "$jaeger_name" "allInOne" "07"



start_test "streaming-with-autoprovisioning-autoscale"
if [ $IS_OPENSHIFT = true ]; then
    # Remove the installation of the operator
    rm ./00-install.yaml ./00-assert.yaml
    REPLICAS=1
else
    REPLICAS=3
fi
export REPLICAS

render_install_elasticsearch "01"

jaeger_name="auto-provisioned"
# Change the resource limits for the autoprovisioned deployment
$YQ e -i '.spec.ingester.resources.requests.memory="20Mi"' ./02-install.yaml
$YQ e -i '.spec.ingester.resources.requests.memory="500m"' ./02-install.yaml

# Enable autoscale
$YQ e -i '.spec.ingester.autoscale=true' ./02-install.yaml
$YQ e -i '.spec.ingester.minReplicas=1' ./02-install.yaml
$YQ e -i '.spec.ingester.maxReplicas=5' ./02-install.yaml

# Assert the autoprovisioned Kafka deployment
$GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./02-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./03-assert.yaml
$GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./04-assert.yaml

# Create the tracegen deployment
# Deploy Tracegen instance to generate load in the Jaeger collector
tracegen_replicas="1"
if [ $IS_OPENSHIFT!="true" ]; then
    tracegen_replicas="3"
fi
render_install_tracegen "$jaeger_name" "$tracegen_replicas" "06"
