#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "examples-agent-as-daemonset"
example_name="agent-as-daemonset"

prepare_daemonset "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-business-application-injected-sidecar"
example_name="simplest"
cat $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./livenessProbe.yaml > ./00-install.yaml
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-service-types"
example_name="service-types"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


if [ "$SKIP_ES_EXTERNAL" = true ]; then
    skip_test "examples-simple-prod" "This test requires an external Elasticsearch instance"
else
    start_test "examples-simple-prod"
    example_name="simple-prod"
    render_install_elasticsearch "00"
    render_install_example "$example_name" "01"
    render_smoke_test_example "$example_name" "02"
fi


if [ "$SKIP_ES_EXTERNAL" = true ]; then
    skip_test "examples-simple-prod-with-volumes" "This test requires an external Elasticsearch instance"
else
    start_test "examples-simple-prod-with-volumes"
    example_name="simple-prod-with-volumes"
    render_install_elasticsearch "00"
    render_install_example "$example_name" "01"
    render_smoke_test_example "$example_name" "02"
fi


start_test "examples-simplest"
example_name="simplest"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-badger"
example_name="with-badger"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-badger-and-volume"
example_name="with-badger-and-volume"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-cassandra"
example_name="with-cassandra"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-with-sampling"
export example_name="with-sampling"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"

###############################################################################
# OpenShift examples ##########################################################
###############################################################################

if [ $IS_OPENSHIFT = true ]; then
    start_test "examples-openshift-agent-as-daemonset"
    $GOMPLATE -f $EXAMPLES_DIR/openshift/hostport-scc-daemonset.yaml -o 00-install.yaml
    $GOMPLATE -f $EXAMPLES_DIR/openshift/service_account_jaeger-agent-daemonset.yaml -o 01-install.yaml
    $GOMPLATE -f $EXAMPLES_DIR/openshift/agent-as-daemonset.yaml -o 03-install.yaml
    render_install_vertx "04"
    $YQ e -i '.spec.template.spec.containers[0].env=[{"name": "JAEGER_AGENT_HOST", "valueFrom": {"fieldRef": {"apiVersion": "v1", "fieldPath": "status.hostIP"}}}]' ./04-install.yaml
    render_find_service "agent-as-daemonset" "production" "order" "00" "05"
else
    skip_test "examples-openshift-agent-as-daemonset" "This test is only supported in OpenShift"
fi
