#!/bin/bash

source $(dirname "$0")/../render-utils.sh

if [ "$IS_OPENSHIFT" != true ]; then
    skip_test "es-increasing-replicas" "Test supported only in OpenShift"
else
    jaeger_name="simple-prod"
    start_test "es-increasing-replicas"

    # Install a Jaeger instance with autoprovisioned ES
    render_install_jaeger "$jaeger_name" "production_autoprovisioned" "00"

    # Increase the number of replicas for the collector, query and ES
    cp ./00-install.yaml ./01-install.yaml
    $YQ e -i '.spec.collector.replicas=2' ./01-install.yaml
    $YQ e -i '.spec.query.replicas=2' ./01-install.yaml
    $YQ e -i '.spec.storage.elasticsearch.nodeCount=2' ./01-install.yaml

    # Check everything was scaled as expected
    cp ./00-assert.yaml ./01-assert.yaml
    $YQ e -i '.spec.replicas=2' ./01-assert.yaml
    $YQ e -i '.status.readyReplicas=2' ./01-assert.yaml

    render_smoke_test "$jaeger_name" "production" "03"
fi



start_test "es-index-cleaner"
jaeger_name="test-es-index-cleaner-with-prefix"
cronjob_name="test-es-index-cleaner-with-prefix-es-index-cleaner"

# Install Elasticsearch instance
render_install_elasticsearch "00"

# Create and assert the Jaeger instance with index cleaner "*/1 * * * *"
render_install_jaeger "$jaeger_name" "production" "01"
$YQ e -i '.spec.storage.options.es.index-prefix=""' ./01-install.yaml
$YQ e -i '.spec.storage.esIndexCleaner.enabled=false' ./01-install.yaml
$YQ e -i '.spec.storage.esIndexCleaner.numberOfDays=0' ./01-install.yaml
$YQ e -i '.spec.storage.esIndexCleaner.schedule="*/1 * * * *"' ./01-install.yaml

# Report some spans
render_report_spans "$JAEGER_NAME" "production" 5 "00" true 02

# Enable Elasticsearch index cleaner
sed "s~enabled: false~enabled: true~gi" ./01-install.yaml > ./03-install.yaml

# Wait for the execution of the cronjob
CRONJOB_NAME=$cronjob_name \
    $GOMPLATE -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
    -o ./04-wait-es-index-cleaner.yaml

# Disable Elasticsearch index cleaner to ensure it is not run again while the test does some checks
$GOMPLATE -f ./01-install.yaml -o ./05-install.yaml

# Check if the indexes were cleaned or not
render_check_indices "false" \
    "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-count-indices', '0'," \
    "00" "06"



if [ "$IS_OPENSHIFT" = "true" ]; then
    start_test "es-multiinstance"
    jaeger_name="instance-1"
    render_install_jaeger "$jaeger_name" "production_autoprovisioned" "01"
else
    skip_test "es-multiinstance" "This test is only supported in OpenShift"
fi


start_test "es-rollover"
export jaeger_name="my-jaeger"

# Install Elasticsearch instance
render_install_elasticsearch "00"

# Install Jaeger
render_install_jaeger "$jaeger_name" "production" "01"

# Report some spans
render_report_spans "$jaeger_name" "production" 2 "00" "true" "02"

# Check the effects in the database
render_check_indices "false" "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-exist'," "00" "03"
render_check_indices "false" "'--pattern', 'jaeger-span-\d{6}', '--assert-count-indices', '0'," "01" "04"

# Step 5 enables rollover. No autogenerated

# Report more spans
render_report_spans "$jaeger_name" "production" "2" "02" "true" "06"

# Check the effects in the database
render_check_indices "false" "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}', '--assert-exist'," "02" "07"
render_check_indices "false" "'--pattern', 'jaeger-span-\d{6}', '--assert-exist'," "03" "08"
render_check_indices "false" "'--name', 'jaeger-span-read', '--assert-exist'," "04" "09"

# Report more spans
render_report_spans "$jaeger_name" "production" "2" "03" "true" "10"

# Wait for the execution of the cronjob
CRONJOB_NAME="my-jaeger-es-rollover" \
    $GOMPLATE \
    -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
    -o ./11-wait-rollover.yaml

# Check the effects in the database
render_check_indices "false" "'--name', 'jaeger-span-000002'," "05" "11"
render_check_indices "false" "'--name', 'jaeger-span-read', '--assert-count-docs', '4', '--jaeger-service', 'smoke-test-service'," "06" "12"


if [ "$IS_OPENSHIFT" = true ]; then
    skip_test "es-spark-dependencies" "This test is not supported in OpenShift"
else
    start_test "es-spark-dependencies"
    render_install_elasticsearch "00"

    # The step 1 creates the Jaeger instance

    CRONJOB_NAME="my-jaeger-spark-dependencies" \
        $GOMPLATE \
            -f $TEMPLATES_DIR/wait-for-cronjob-execution.yaml.template \
            -o ./02-wait-spark-job.yaml
fi


if [ "$IS_OPENSHIFT" != true ]; then
    skip_test "es-streaming-autoprovisioned" "This test is only supported in OpenShift"
else
    start_test "es-streaming-autoprovisioned"

    export CLUSTER_NAME="auto-provisioned"
    export REPLICAS=1
    $GOMPLATE -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template -o ./00-assert.yaml
    $GOMPLATE -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f $TEMPLATES_DIR/assert-entity-operator.yaml.template -o ./02-assert.yaml

    render_smoke_test "auto-provisioned" "allInOne" "03"
fi
