#!/bin/bash

source $(dirname "$0")/../render-utils.sh

if [ $IS_OPENSHIFT = true ]; then
    skip_test "cassandra-spark" "Test not supported in OpenShift"
else
    start_test "cassandra-spark"
    # Create Cassandra instance and assert it
    render_install_cassandra "00"
    # Create the Jaeger instance
    export JAEGER_NAME=test-spark-deps
    export DEP_SCHEDULE=true
    export CASSANDRA_MODE=prod
    $GOMPLATE -f $TEMPLATES_DIR/cassandra-jaeger-install.yaml.template -o ./01-install.yaml

    export CRONJOB_APIVERSION
    if version_gt $KUBE_VERSION "1.24"; then
        CRONJOB_APIVERSION="batch/v1"
    else
        CRONJOB_APIVERSION="batch/v1beta1"
    fi
    $GOMPLATE -f ./01-assert.yaml.template -o ./01-assert.yaml
fi



start_test "collector-autoscale"
jaeger_name="simple-prod"

if [ $IS_OPENSHIFT!="true" ]; then
    render_install_elasticsearch "00"
fi

ELASTICSEARCH_NODECOUNT="1"
render_install_jaeger "$jaeger_name" "production" "01"
# Change the resource limits for the Jaeger deployment
$YQ e -i '.spec.collector.resources.requests.memory="20Mi"' 01-install.yaml
$YQ e -i '.spec.collector.resources.requests.memory="300m"' 01-install.yaml

# Enable autoscale
$YQ e -i '.spec.collector.autoscale=true' 01-install.yaml
$YQ e -i '.spec.collector.minReplicas=1' 01-install.yaml
$YQ e -i '.spec.collector.maxReplicas=3' 01-install.yaml

# Deploy Tracegen instance to generate load in the Jaeger collector
render_install_tracegen "$jaeger_name" "02"



# Helper function to generate the same tests multiple times but with different
# reporting protocols
function generate_otlp_e2e_tests() {
    test_protocol=$1

    if [ "$IS_OPENSHIFT" = "true" ]; then
        is_secured="true"
    else
        is_secured="false"
    fi

    start_test "collector-otlp-allinone-$test_protocol"
    render_install_jaeger "my-jaeger" "allInOne" "00"
    render_otlp_smoke_test "my-jaeger" "$test_protocol" "$is_secured" "01"

    start_test "collector-otlp-production-$test_protocol"
    render_install_elasticsearch "00"
    render_install_jaeger "my-jaeger" "production" "01"
    render_otlp_smoke_test "my-jaeger" "$test_protocol" "$is_secured" "02"
}

generate_otlp_e2e_tests "http"
generate_otlp_e2e_tests "grpc"



if [ $IS_OPENSHIFT = true ]; then
    skip_test "istio" "Test not supported in OpenShift"
else
    start_test "istio"
    export jaeger_name="simplest"
    cat $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./livelinessprobe.template > ./03-install.yaml
    render_find_service "$jaeger_name" "allInOne" "order" "00" "04"

    # One of the first steps of this test is enabling the Istio sidecar injection
    # for the namespace. That means, each pod is started will have an Istio sidecar.
    # A Job is not considered complete until all containers have stopped running.
    # Istio Sidecars run indefinitely. So, when a job is started, the sidecar will
    # stay forever there and the test will be marked as failed. Also, since the
    # container job starts and finish, the pod status will be `NotReady`.
    # Stopping Istio from the pod doing a POST HTTP query to
    # http://localhost:15000/quitquitquit (endpoint available since Istio 1.3),
    # solves the issue
    patched_file="./04-find-service.yaml"
    $YQ e -i '.spec.template.spec.containers[0].command = ["/bin/sh","-c"]' $patched_file
    $YQ e -i '.spec.template.spec.containers[0].args= ["./query && curl -sf -XPOST http://localhost:15000/quitquitquit"]' $patched_file
fi


if [ $IS_OPENSHIFT = true ]; then
    skip_test "outside-cluster" "Test not supported in OpenShift"
else
    start_test "outside-cluster"
    jaeger_name="my-jaeger"
    render_install_elasticsearch "00"
    render_install_jaeger "$jaeger_name" "production" "01"
    $GOMPLATE -f ./03-check-collector.yaml.template -o 03-check-collector.yaml
fi
