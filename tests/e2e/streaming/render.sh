#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "streaming-simple"
render_install_kafka "my-cluster" "00"
render_install_elasticsearch "01"
JAEGER_NAME="simple-streaming" $GOMPLATE -f $TEMPLATES_DIR/streaming-jaeger-assert.yaml.template -o ./04-assert.yaml
render_smoke_test "simple-streaming" "production" "05"


start_test "streaming-with-tls"
render_install_kafka "my-cluster" "00"
render_install_elasticsearch "03"
render_smoke_test "tls-streaming" "production" "05"



start_test "streaming-with-autoprovisioning"
jaeger_name="auto-provisioned"

if [ $IS_OPENSHIFT = true ]; then
    # Remove the installation of the operator
    rm ./00-install.yaml ./00-assert.yaml
fi

render_install_elasticsearch "01"
render_assert_kafka "true" "$jaeger_name" "03"
render_smoke_test "$jaeger_name" "production" "07"



start_test "streaming-with-autoprovisioning-autoscale"
if [ $IS_OPENSHIFT = true ]; then
    # Remove the installation of the operator
    rm ./00-install.yaml ./00-assert.yaml
fi

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
render_assert_kafka "true" "$jaeger_name" "03"

# Create the tracegen deployment
# Deploy Tracegen instance to generate load in the Jaeger collector
render_install_tracegen "$jaeger_name" "3" "06"
