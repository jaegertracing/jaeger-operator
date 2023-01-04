#!/bin/bash

source $(dirname "$0")/../render-utils.sh

is_secured="false"
if [ $IS_OPENSHIFT = true ]; then
    is_secured="true"
fi


###############################################################################
# TEST NAME: es-from-aio-to-production
# DESCRIPTION: allInOne to production
###############################################################################
start_test "es-from-aio-to-production"
jaeger_name="my-jaeger"
render_install_jaeger "$jaeger_name" "allInOne" "00"
render_smoke_test "$jaeger_name" "$is_secured" "01"

jaeger_deploy_mode="production"
if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    jaeger_deploy_mode="production_autoprovisioned"
else
    render_install_elasticsearch "upstream" "02"
fi
render_install_jaeger "$jaeger_name" "$jaeger_deploy_mode" "03"
if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    # when we deploy the jaeger CR, irrespective of deployment strategy "normalizeElasticsearch" func called.
    # It adds the default parameters. as a result in the generated CR: redundancyPolicy is SingleRedundancy and node count is 3
    # normalizeElasticsearch: https://github.com/jaegertracing/jaeger-operator/blob/2ccf2d4a4ab799ba07a4c408bde8a2caad3d16f8/pkg/strategy/controller.go#L132
    # later we are switching from allInOne to production, we include node count as 1, but not touched redundancyPolicy.
    # Hence creates Elasticsearch CR with SingleRedundancy and node count 1. Which is invalid and failed to deploy the elasticsearch node.
    # no idea, is this issue with product or kuttl test?
    # as a workaround updating redundancyPolicy explicitly.
    $YQ e -i '.spec.storage.elasticsearch.redundancyPolicy="ZeroRedundancy"' ./03-install.yaml
fi
render_smoke_test "$jaeger_name" "$is_secured" "04"


###############################################################################
# TEST NAME: es-increasing-replicas
###############################################################################
start_test "es-increasing-replicas"
jaeger_name="simple-prod"

if [ $IS_OPENSHIFT = true ]; then
    # For OpenShift, we want to test changes in the Elasticsearch instances
    # autoprovisioned by the Elasticsearch OpenShift Operator
    jaeger_deployment_mode="production_autoprovisioned"
else
    jaeger_deployment_mode="production"
    render_install_elasticsearch "upstream" "00"
fi
render_install_jaeger "$jaeger_name" "$jaeger_deployment_mode" "01"

# Increase the number of replicas for the collector and query
cp ./01-install.yaml ./02-install.yaml
$YQ e -i '.spec.collector.replicas=2' ./02-install.yaml
$YQ e -i '.spec.query.replicas=2' ./02-install.yaml

# Check everything was scaled as expected
cp ./01-assert.yaml ./02-assert.yaml
$YQ e -i '.spec.replicas=2' ./02-assert.yaml
$YQ e -i '.status.readyReplicas=2' ./02-assert.yaml

render_smoke_test "$jaeger_name" "$is_secured" "03"

if [ $IS_OPENSHIFT = true ]; then
    # Increase the number of nodes for autoprovisioned ES
    cp ./02-install.yaml ./04-install.yaml
    $YQ e -i '.spec.storage.elasticsearch.nodeCount=2' ./04-install.yaml
    $GOMPLATE -f ./openshift-check-es-nodes.yaml.template -o ./05-check-es-nodes.yaml
fi


###############################################################################
# TEST NAME: es-index-cleaner-*
###############################################################################
# Helper function to render the ES index cleaner E2E test using different
# deployment modes
function es_index_cleaner(){
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for es_index_cleaner. Usage: es_index_cleaner <test postfix name> <Jaeger deployment mode>"
        exit 1
    fi
    postfix=$1
    jaeger_deployment_strategy=$2

    start_test "es-index-cleaner$postfix"
    jaeger_name="test-es-index-cleaner-with-prefix"
    cronjob_name="test-es-index-cleaner-with-prefix-es-index-cleaner"
    secured_es_connection="false"

    if [ "$jaeger_deployment_strategy" = "production" ]; then
        # Install Elasticsearch instance
        render_install_elasticsearch "upstream" "00"
        ELASTICSEARCH_URL="http://elasticsearch"
    elif [ "$jaeger_deployment_strategy" = "production_managed_es" ]; then
        render_install_elasticsearch "openshift_operator" "00"
        secured_es_connection="true"
    else
        ELASTICSEARCH_URL="https://elasticsearch"
        secured_es_connection="true"
    fi

    cp ../../es-index-cleaner-upstream/* .

    # Create and assert the Jaeger instance with index cleaner "*/1 * * * *"
    render_install_jaeger "$jaeger_name" "$jaeger_deployment_strategy" "01"
    $YQ e -i '.spec.storage.options.es.index-prefix=""' ./01-install.yaml
    $YQ e -i '.spec.storage.esIndexCleaner.enabled=false' ./01-install.yaml
    $YQ e -i '.spec.storage.esIndexCleaner.numberOfDays=0' ./01-install.yaml
    $YQ e -i '.spec.storage.esIndexCleaner.schedule="*/1 * * * *"' ./01-install.yaml

    # Report some spans
    render_report_spans "$JAEGER_NAME" "$is_secured" "5" "00" "true" "02"

    # Enable Elasticsearch index cleaner
    sed "s~enabled: false~enabled: true~gi" ./01-install.yaml > ./03-install.yaml

    # Wait for the execution of the cronjob
    CRONJOB_NAME=$cronjob_name \
        $GOMPLATE -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
        -o ./04-wait-es-index-cleaner.yaml

    # Disable Elasticsearch index cleaner to ensure it is not run again while the test does some checks
    $GOMPLATE -f ./01-install.yaml -o ./05-install.yaml

    # Check if the indexes were cleaned or not
    render_check_indices "$secured_es_connection" \
        "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-count-indices', '0'," \
        "00" "06"
}

if [ "$SKIP_ES_EXTERNAL" = true ]; then
    skip_test "es-index-cleaner-upstream" "SKIP_ES_EXTERNAL is true"
else
    es_index_cleaner "-upstream" "production"
fi

if [ "$IS_OPENSHIFT" = true ]; then
    es_index_cleaner "-autoprov" "production_autoprovisioned"
else
    skip_test "es-index-cleaner-autoprov" "Test only supported in OpenShift"
fi


if [ "$IS_OPENSHIFT" = true ]; then
    get_elasticsearch_openshift_operator_version
    if [ -n "$(version_ge "$ESO_OPERATOR_VERSION" "5.4")" ]; then
        es_index_cleaner "-managed" "production_managed_es"
    else
        skip_test "es-index-cleaner-managed" "Test only supported with Elasticsearch OpenShift Operator >= 5.4"
    fi
else
    skip_test "es-index-cleaner-managed" "Test only supported in OpenShift"
fi


###############################################################################
# TEST NAME: es-multiinstance
###############################################################################
if [ "$IS_OPENSHIFT" = true ]; then
    start_test "es-multiinstance"
    jaeger_name="instance-1"
    render_install_jaeger "$jaeger_name" "production_autoprovisioned" "01"
    $GOMPLATE -f ./03-create-second-instance.yaml.template -o 03-create-second-instance.yaml
else
    skip_test "es-multiinstance" "This test is only supported in OpenShift"
fi


###############################################################################
# TEST NAME: es-rollover-*
###############################################################################
# Helper function to render the ES Rollover E2E test using different
# deployment modes
function es_rollover(){
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for es_rollover. Usage: es_rollover <test postfix name> <Jaeger deployment mode>"
        exit 1
    fi
    postfix=$1
    jaeger_deployment_strategy=$2

    start_test "es-rollover$postfix"

    cp ../../es-rollover-upstream/* .

    jaeger_name="my-jaeger"
    secured_es_connection="false"

    if [ "$jaeger_deployment_strategy" = "production" ]; then
        # Install Elasticsearch instance
        render_install_elasticsearch "upstream" "00"
        ELASTICSEARCH_URL="http://elasticsearch"
    elif [ "$jaeger_deployment_strategy" = "production_managed_es" ]; then
        render_install_elasticsearch "openshift_operator" "00"
        secured_es_connection="true"
    else
        ELASTICSEARCH_URL="https://elasticsearch"
        secured_es_connection="true"
    fi


    # Install Jaeger
    render_install_jaeger "$jaeger_name" "$jaeger_deployment_strategy" "01"

    # Report some spans
    render_report_spans "$jaeger_name" "$is_secured" "2" "00" "true" "02"

    # Check the effects in the database
    render_check_indices "$secured_es_connection" "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-exist'," "00" "03"
    render_check_indices "$secured_es_connection" "'--pattern', 'jaeger-span-\d{6}', '--assert-count-indices', '0'," "01" "04"

    # Step 5 enables rollover. No autogenerated

    # Report more spans
    render_report_spans "$jaeger_name" "$is_secured" "2" "02" "true" "06"

    # Check the effects in the database
    render_check_indices "$secured_es_connection" "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-exist'," "02" "07"
    render_check_indices "$secured_es_connection" "'--pattern', 'jaeger-span-\d{6}', '--assert-exist'," "03" "08"
    render_check_indices "$secured_es_connection" "'--name', 'jaeger-span-read', '--assert-exist'," "04" "09"

    # Report more spans
    render_report_spans "$jaeger_name" "$is_secured" "2" "03" "true" "10"

    # Wait for the execution of the cronjob
    CRONJOB_NAME="my-jaeger-es-rollover" \
        $GOMPLATE \
        -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
        -o ./11-wait-rollover.yaml

    # Check the effects in the database
    render_check_indices "$secured_es_connection" "'--name', 'jaeger-span-000002'," "05" "11"
    render_check_indices "$secured_es_connection" "'--name', 'jaeger-span-read', '--assert-count-docs', '4', '--jaeger-service', 'smoke-test-service'," "06" "12"
}

if [ "$SKIP_ES_EXTERNAL" = true ]; then
    skip_test "es-rollover-upstream" "SKIP_ES_EXTERNAL is true"
else
    es_rollover "-upstream" "production"
fi

if [ "$IS_OPENSHIFT" = true ]; then
    es_rollover "-autoprov" "production_autoprovisioned"
else
    skip_test "es-rollover-autoprov" "Test only supported in OpenShift"
fi

if [ "$IS_OPENSHIFT" = true ]; then
    get_elasticsearch_openshift_operator_version
    if [ -n "$(version_ge "$ESO_OPERATOR_VERSION" "5.4")" ]; then
        es_rollover "-managed" "production_managed_es"
    else
        skip_test "es-rollover-managed" "Test only supported with Elasticsearch OpenShift Operator >= 5.4"
    fi
else
    skip_test "es-rollover-managed" "Test only supported in OpenShift"
fi


###############################################################################
# TEST NAME: es-spark-dependencies
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    skip_test "es-spark-dependencies" "This test is not supported in OpenShift"
else
    start_test "es-spark-dependencies"
    render_install_elasticsearch "upstream" "00"

    # The step 1 creates the Jaeger instance

    CRONJOB_NAME="my-jaeger-spark-dependencies" \
        $GOMPLATE \
            -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
            -o ./02-wait-spark-job.yaml
fi


###############################################################################
# TEST NAME: es-streaming-autoprovisioned
###############################################################################
if [[ $IS_OPENSHIFT = true && $SKIP_KAFKA = false ]]; then
    start_test "es-streaming-autoprovisioned"
    jaeger_name="auto-provisioned"

    render_assert_kafka "true" "$jaeger_name" "00"
    render_smoke_test "$jaeger_name" "true" "04"
else
    skip_test "es-streaming-autoprovisioned" "This test is only supported in OpenShift with SKIP_KAFKA is false"
fi
