#!/bin/bash

source $(dirname "$0")/../render-utils.sh

is_secured="false"
if [ $IS_OPENSHIFT = true ]; then
    is_secured="true"
fi


###############################################################################
# TEST NAME: streaming-simple
###############################################################################
if [ $SKIP_KAFKA = true ]; then
    skip_test "streaming-simple" "SKIP_KAFKA is true"
else
    start_test "streaming-simple"
    render_install_kafka "my-cluster" "00"
    render_install_elasticsearch "upstream" "03"
    JAEGER_NAME="simple-streaming" $GOMPLATE -f $TEMPLATES_DIR/streaming-jaeger-assert.yaml.template -o ./04-assert.yaml
    render_smoke_test "simple-streaming" "$is_secured" "05"
fi


###############################################################################
# TEST NAME: streaming-with-tls
###############################################################################
if [ $SKIP_KAFKA = true ]; then
    skip_test "streaming-with-tls" "SKIP_KAFKA is true"
else
    start_test "streaming-with-tls"
    render_install_kafka "my-cluster" "00"
    render_install_elasticsearch "upstream" "03"
    render_smoke_test "tls-streaming" "$is_secured" "05"
fi


###############################################################################
# TEST NAME: streaming-with-autoprovisioning-autoscale
###############################################################################
if [ $SKIP_KAFKA = true ]; then
    skip_test "streaming-with-autoprovisioning-autoscale" "SKIP_KAFKA is true"
else
    start_test "streaming-with-autoprovisioning-autoscale"
    if [ $KAFKA_OLM = true ]; then
        # Remove the installation of the operator
        rm ./00-install.yaml ./00-assert.yaml
    fi

    render_install_elasticsearch "upstream" "01"

    jaeger_name="auto-provisioned"
    # Change the resource limits for the autoprovisioned deployment
    $YQ e -i '.spec.ingester.resources.requests.memory="20Mi"' ./02-install.yaml
    $YQ e -i '.spec.ingester.resources.requests.memory="500m"' ./02-install.yaml

    # Enable autoscale
    $YQ e -i '.spec.ingester.autoscale=true' ./02-install.yaml
    $YQ e -i '.spec.ingester.minReplicas=1' ./02-install.yaml
    $YQ e -i '.spec.ingester.maxReplicas=2' ./02-install.yaml

    # Assert the autoprovisioned Kafka deployment
    render_assert_kafka "true" "$jaeger_name" "03"

    if kubectl api-versions | grep "autoscaling/v2beta2" -q; then
        # Use the autoscaling/v2beta2 file
        rm ./07-assert.yaml
    else
        # Use the autoscaling/v2 file
        rm ./08-assert.yaml
    fi
fi
