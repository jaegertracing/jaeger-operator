#!/bin/bash

source $(dirname "$0")/../render-utils.sh

# ###############################################################################
# # TEST NAME: cassandra-spark
# ###############################################################################
# if [ $IS_OPENSHIFT = true ]; then
#     skip_test "cassandra-spark" "Test not supported in OpenShift"
# else
#     start_test "cassandra-spark"
#     # Create Cassandra instance and assert it
#     render_install_cassandra "00"
#     # Create the Jaeger instance
#     export JAEGER_NAME=test-spark-deps
#     export DEP_SCHEDULE=true
#     export CASSANDRA_MODE=prod
#     $GOMPLATE -f $TEMPLATES_DIR/cassandra-jaeger-install.yaml.template -o ./01-install.yaml

#     export CRONJOB_APIVERSION
#     if version_gt $KUBE_VERSION "1.24"; then
#         CRONJOB_APIVERSION="batch/v1"
#     else
#         CRONJOB_APIVERSION="batch/v1beta1"
#     fi
#     $GOMPLATE -f ./01-assert.yaml.template -o ./01-assert.yaml
# fi


###############################################################################
# TEST NAME: collector-autoscale
###############################################################################
start_test "collector-autoscale"
jaeger_name="simple-prod"
jaeger_deploy_mode="production"

if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    jaeger_deploy_mode="production_autoprovisioned"
else
    render_install_elasticsearch "upstream" "00"
fi

ELASTICSEARCH_NODECOUNT="1"
render_install_jaeger "$jaeger_name" "$jaeger_deploy_mode" "01"
# Change the resource limits for the Jaeger deployment
$YQ e -i '.spec.collector.resources.requests.memory="200m"' 01-install.yaml

# Enable autoscale
$YQ e -i '.spec.collector.autoscale=true' 01-install.yaml
$YQ e -i '.spec.collector.minReplicas=1' 01-install.yaml
$YQ e -i '.spec.collector.maxReplicas=2' 01-install.yaml

if version_lt $KUBE_VERSION "1.23"; then
    # Use the autoscaling/v2beta2 file
    rm ./02-assert.yaml
else
    # Use the autoscaling/v2 file
    rm ./03-assert.yaml
fi

###############################################################################
# TEST NAME: collector-otlp-*
###############################################################################
# Helper function to generate the same tests multiple times but with different
# reporting protocols
function generate_otlp_e2e_tests() {
    test_protocol=$1

    is_secured="false"
    if [ "$IS_OPENSHIFT" = true ]; then
        is_secured="true"
    fi

    # TEST NAME: collector-otlp-allinone-*
    start_test "collector-otlp-allinone-$test_protocol"
    render_install_jaeger "my-jaeger" "allInOne" "00"
    render_otlp_smoke_test "my-jaeger" "$test_protocol" "$is_secured" "01"

    # TEST NAME: collector-otlp-production-*
    start_test "collector-otlp-production-$test_protocol"
    jaeger_deploy_mode="production"
    if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
        jaeger_deploy_mode="production_autoprovisioned"
    else
        render_install_elasticsearch "upstream" "00"
    fi
    render_install_jaeger "my-jaeger" "$jaeger_deploy_mode" "01"
    render_otlp_smoke_test "my-jaeger" "$test_protocol" "$is_secured" "02"
}

generate_otlp_e2e_tests "http"
generate_otlp_e2e_tests "grpc"


###############################################################################
# TEST NAME: istio
###############################################################################
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


###############################################################################
# TEST NAME: outside-cluster
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    skip_test "outside-cluster" "Test not supported in OpenShift"
else
    start_test "outside-cluster"
    jaeger_name="my-jaeger"
    render_install_elasticsearch "upstream" "00"
    render_install_jaeger "$jaeger_name" "production" "01"
    $GOMPLATE -f ./03-check-collector.yaml.template -o 03-check-collector.yaml
fi


###############################################################################
# TEST NAME: set-custom-img
###############################################################################
start_test "set-custom-img"
jaeger_name="my-jaeger"
jaeger_deploy_mode="production"
if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    jaeger_deploy_mode="production_autoprovisioned"
else
    render_install_elasticsearch "upstream" "00"
fi
render_install_jaeger "$jaeger_name" "$jaeger_deploy_mode" "01"
cp ./01-install.yaml ./02-install.yaml
$YQ e -i '.spec.collector.image="test"' ./02-install.yaml


###############################################################################
# TEST NAME: non-cluster-wide
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    skip_test "non-cluster-wide" "Test not supported in OpenShift"
else
    start_test "non-cluster-wide"
    $GOMPLATE -f ./00-undeploy.yaml.template -o 00-undeploy.yaml
    $GOMPLATE -f ./01-install.yaml.template -o 01-install.yaml
    jaeger_name="my-jaeger"
    render_install_jaeger "$jaeger_name" "allInOne" "02"
    $GOMPLATE -f ./03-install.yaml.template -o 03-install.yaml
fi
