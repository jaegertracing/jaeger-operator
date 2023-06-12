#!/bin/bash
#
# Utils for the render.sh scripts.
#
if [[ "$(basename -- "$0")" = "render-utils.sh" ]]; then
    error "Don't run $0, source it" >&2
    exit 1
fi

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../)
source $ROOT_DIR/hack/common.sh


###############################################################################
# Functions ###################################################################
###############################################################################

# Render a smoke test.
#   render_smoke_test <jaeger_instance_name> <is_secured> <test_step>
#
# Example:
#   render_smoke_test "my-jaeger" "true" "01"
# Generates the `01-smoke-test.yaml` and `01-assert.yaml` files. A smoke test
# will be run against the Jaeger instance called `my-jaeger`. The query service
# is using security.
function render_smoke_test() {
    if [ "$#" -ne 3 ]; then
        error "Wrong number of parameters used for render_smoke_test. Usage: render_smoke_test <jaeger_instance_name> <is_secured> <test_step>"
        exit 1
    fi

    jaeger=$1
    is_secured=$2
    test_step=$3

    if [ $is_secured = true ]; then
        protocol="https://"
        query_port=":443"
        template="$TEMPLATES_DIR/openshift/smoke-test.yaml.template"
    elif [ $is_secured = false ]; then
        protocol="http://"
        query_port=":16686"
        template="$TEMPLATES_DIR/smoke-test.yaml.template"
    else
        error "$is_secured value is invalid for render_smoke_test. Only true|false accepted"
        exit 1
    fi

    export JAEGER_QUERY_ENDPOINT="$protocol$jaeger-query$query_port"
    export JAEGER_COLLECTOR_ENDPOINT="http://$jaeger-collector-headless:14268"
    export JAEGER_NAME=$jaeger

    $GOMPLATE -f $template -o ./$test_step-smoke-test.yaml
    $GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./$test_step-assert.yaml

    unset JAEGER_NAME
    unset JAEGER_QUERY_ENDPOINT
    unset JAEGER_COLLECTOR_ENDPOINT
}

# Render a smoke test using an OTLP client.
#   render_otlp_smoke_test <jaeger_instance_name> <protocol> <is_secured> <test_step>
#
# Example:
#   render_otlp_smoke_test "my-jaeger" "http" "true" "01"
# Generates the `01-smoke-test.yaml` and `01-assert.yaml` files. A smoke test
# will be run against the Jaeger instance called `my-jaeger`. It will use HTTP to
# report the traces.
# Accepted values for <protocol>:
#   * http: use HTTP to report the traces
#   * grpc: use GRPC to report the traces
function render_otlp_smoke_test() {
    if [ "$#" -ne 4 ]; then
        error "Wrong number of parameters used for render_otlp_smoke_test. Usage: render_otlp_smoke_test <jaeger_instance_name> <protocol> <is_secured> <test_step>"
        exit 1
    fi

    jaeger=$1
    reporting_protocol=$2
    is_secured=$3
    test_step=$4

    if [ $is_secured = true ]; then
        protocol="https://"
        query_port=":443"
        template="$TEMPLATES_DIR/openshift/otlp-smoke-test.yaml.template"
    else
        protocol="http://"
        query_port=":16686"
        template="$TEMPLATES_DIR/otlp-smoke-test.yaml.template"
    fi

    if [ "$reporting_protocol" = "grpc" ]; then
        reporting_port=":4317"
    else
        reporting_port=":4318"
    fi

    export JAEGER_QUERY_ENDPOINT="$protocol$jaeger-query$query_port"
    export OTEL_EXPORTER_OTLP_ENDPOINT="http://$jaeger-collector-headless$reporting_port"
    export JAEGER_NAME=$jaeger

    REPORTING_PROTOCOL=$reporting_protocol $GOMPLATE -f $template -o ./$test_step-smoke-test.yaml
    $GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./$test_step-assert.yaml

    unset JAEGER_NAME
    unset JAEGER_QUERY_ENDPOINT
    unset OTEL_EXPORTER_OTLP_ENDPOINT
}

# Render a reporting spans.
#   render_report_spans <jaeger_instance_name> <is_secured> <spans> <job_number> <ensure_reported_spans> <test_step>
#
# Example:
#   render_report_spans "my-jaeger" "true" "10" "01" "true" "02"
# Generates the `02-report-spans.yaml` and `02-assert.yaml` files, that will
# start the report-spans-01 job. Report spans to the Jaeger instance. If
# `ensure_reported_spans` is `true`, the job will check the spans were generated
# properly. It will use a secure connection.
# Note: If OpenShift and <ensure_reported_spans> is `true`, other files will be
# generated during the test execution to get the needed OAuth tokens to query
# the Jaeger REST API.
function render_report_spans() {
    if [ "$#" -ne 6 ]; then
        error "Wrong number of parameters used for render_report_spans. Usage: render_report_spans <jaeger_instance_name> <is_secured>  <spans> <services> <job_number> <ensure_reported_spans> <test_step>"
        exit 1
    fi

    jaeger=$1
    is_secured=$2
    number_of_spans=$3
    job_number=$4
    ensure_reported_spans=$5
    test_step=$6

    export JAEGER_NAME=$jaeger
    export JAEGER_COLLECTOR_ENDPOINT="http://$jaeger-collector-headless:14268"
    export JOB_NUMBER=$job_number
    export DAYS=$number_of_spans

    if [ "$is_secured" = true ]; then
        protocol="https://"
        query_port=""
        template=$TEMPLATES_DIR/openshift/report-spans.yaml.template
    else
        protocol="http://"
        query_port=":16686"
        template=$TEMPLATES_DIR/report-spans.yaml.template
    fi

    if [ $ensure_reported_spans = true ]; then
        export ENSURE_REPORTED_SPANS=true
        export JAEGER_QUERY_ENDPOINT="$protocol$jaeger-query$query_port"
    fi

    params=""
    if [ $IS_OPENSHIFT = true ] && [ $ensure_reported_spans = true ] && [ "$deployment_strategy" != "allInOne" ]; then
        params="-t $TEMPLATES_DIR/openshift/configure-api-query-oauth.yaml.template"
    fi

    $GOMPLATE -f $template $params -o ./$test_step-report-spans.yaml
    $GOMPLATE -f $TEMPLATES_DIR/assert-report-spans.yaml.template -o ./$test_step-assert.yaml

    unset JAEGER_COLLECTOR_ENDPOINT
    unset JAEGER_QUERY_ENDPOINT
    unset JOB_NUMBER
    unset DAYS
    unset ENSURE_REPORTED_SPANS
}


# Render a check indices job.
#   render_check_indices <secured>  <cmd_parameters> <job_number> <test_step>
#
#
# Example:
#   render_check_indices "false" "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}'" "01" "02"
# Renders the 02-check-indices.yaml and 02-assert.yaml files. The `01-check-indices` job
# will be started using the CMD parameters. The connection to the ES instance will be done
# without using certificates
function render_check_indices() {
    if [ "$#" -ne 4 ]; then
        error "Wrong number of parameters used for render_check_indices. Usage: render_check_indices <secured>  <cmd_parameters> <job_number> <test_step>"
        exit 1
    fi

    secured=$1
    cmd_parameters=$2
    job_number=$3
    test_step=$4


    escape_command "$cmd_parameters"
    mount_secret=""

    if [ $IS_OPENSHIFT = true ] && [ $secured = true ] ; then
        # The certificates to connect to the database are needed. The asserts
        # container uses the same secret created to be used by the index cleaner
        mount_secret="$JAEGER_NAME-curator"
    fi

    JOB_NUMBER=$job_number \
    CMD_PARAMETERS="$cmd_parameters" \
    MOUNT_SECRET="$mount_secret" \
        $GOMPLATE \
            -f $TEMPLATES_DIR/check-indices.yaml.template \
            -o ./$test_step-check-indices.yaml

    JOB_NUMBER=$job_number \
    $GOMPLATE \
        -f $TEMPLATES_DIR/assert-check-indices.yaml.template \
        -o ./$test_step-assert.yaml
}



# Render the files to install Cassandra database.
#   render_install_cassandra <test_step>
#
# Example:
#   render_install_cassandra "00"
# Generates the `00-install.yaml` and `00-assert.yaml` files. A Cassandra
# instance will be installed.
function render_install_cassandra() {
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for render_install_cassandra. Usage: render_install_cassandra <test_step>"
        exit 1
    fi

    test_step=$1

    $GOMPLATE -f $TEMPLATES_DIR/cassandra-install.yaml.template -o ./$test_step-install.yaml
    $GOMPLATE -f $TEMPLATES_DIR/cassandra-assert.yaml.template -o ./$test_step-assert.yaml
}


# Render the files to install Elasticsearch database.
#   render_install_elasticsearch <deploy_mode> <test_step>
#
# Supported values for <deploy_mode>:
# * upstream: deploy an ES instance using the upstream image.
# * openshift_operator: deploy an external ES instance using the Elasticsearch
#   OpenShift Operator.
# Example:
#   render_install_elasticsearch "upstream" "00"
# Generates the `00-install.yaml` and `00-assert.yaml` files. An Elasticsearch
# instance will be installed. It will use the upstream ES image.
function render_install_elasticsearch() {
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for render_install_elasticsearch. Usage: render_install_elasticsearch <deploy_mode> <test_step>"
        exit 1
    fi

    deploy_mode=$1
    test_step=$2

    if [ "$deploy_mode" = "upstream" ]; then
        if [ "$IS_OPENSHIFT" = true ]; then
            template=$TEMPLATES_DIR/openshift/elasticsearch-install.yaml.template
            $YQ eval -s '"elasticsearch_" + $index' $TEST_DIR/elasticsearch.yml
            $YQ eval -i '.spec.template.spec.serviceAccountName="deploy-elasticsearch"' ./elasticsearch_0.yml
        else
            template=$TEMPLATES_DIR/elasticsearch-install.yaml.template
        fi

        $GOMPLATE -f $template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/elasticsearch-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ "$deploy_mode" = "openshift_operator" ]; then
        if [ "$IS_OPENSHIFT" = true ]; then
            $GOMPLATE -f $TEMPLATES_DIR/openshift/elasticsearch-managed-install.yaml.template -o ./$test_step-install.yaml
            $GOMPLATE -f $TEMPLATES_DIR/openshift/elasticsearch-managed-assert.yaml.template -o ./$test_step-assert.yaml
        else
            error "openshift_operator deploy mode cannot be used if not using an OpenShift cluster"
            exit 1
        fi
    else
        error "$deploy_mode not recognized as deploy_mode for render_install_elasticsearch"
        exit 1
    fi
}


# Render the files to install a Jaeger instance with production strategy.
#   render_install_jaeger <jaeger_instance_name> <deploy_mode> <test_step>
#
# Accepted values for <deploy_mode>:
#   * allInOne: all in one deployment.
#   * production: production using Elasticsearch.
#   * production_cassandra: production using Cassandra.
#   * production_autoprovisioned: production deployment autoprovisioning ES. Only
#       available for OpenShift environments using the Elasticsearch OpenShift Operator.
#   * production_managed_es: production deployment using an external
#       Elasticsearch instance provisioned with the Elasticsearch OpenShift Operator.
#
# Example:
#   render_install_jaeger "my-jaeger" "production" "00"
# Generates the `00-install.yaml` and `00-assert.yaml` files. Production Jaeger
# will be installed. Its name will be `my-jaeger`.
function render_install_jaeger() {
    if [ "$#" -ne 3 ]; then
        error "Wrong number of parameters used for render_install_jaeger. Usage: render_install_jaeger <jaeger_instance_name> <deploy_mode> <test_step>"
        exit 1
    fi

    export JAEGER_NAME=$1
    deploy_mode=$2
    test_step=$3

    if [ $deploy_mode = "allInOne" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ $deploy_mode = "production" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/production-jaeger-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ $deploy_mode = "production_cassandra" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/cassandra-jaeger-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/assert-jaeger-deployment.yaml.template -o ./$test_step-assert.yaml
    elif [ $deploy_mode = "production_autoprovisioned" ]; then
        if [ $IS_OPENSHIFT != "true" ]; then
            error "production_autoprovisioned Jaeger deploy mode is only supported for OpenShift"
            exit 1
        fi
        $GOMPLATE -f $TEMPLATES_DIR/openshift/production-jaeger-autoprovisioned-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ $deploy_mode = "production_managed_es" ]; then
        if [ $IS_OPENSHIFT != "true" ]; then
            error "production_managed_es Jaeger deploy mode is only supported for OpenShift"
            exit 1
        fi
        $GOMPLATE -f $TEMPLATES_DIR/openshift/production-jaeger-autoprovisioned-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    else
        error "Used '$deploy_mode' is not a valid value for <deploy_mode>"
        exit 1
    fi
}


# Run steps needed before running a Jaeger as DaemonSet
#   prepare_daemonset <test_step>
#
# Example:
#   prepare_daemonset "simplest" "00"
# Generates a `00-install.yaml` file with all the steps needed to un a Jaeger
# instance as DaemonSet.
function prepare_daemonset(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for prepare_daemonset. Usage: prepare_daemonset <test_step>"
        exit 1
    fi

    test_step=$1

    if [ $IS_OPENSHIFT = true ]; then
        cat $EXAMPLES_DIR/openshift/hostport-scc-daemonset.yaml > ./$test_step-install.yaml
        echo "---" >> ./$test_step-install.yaml
        cat $EXAMPLES_DIR/openshift/service_account_jaeger-agent-daemonset.yaml >> ./$test_step-install.yaml
    fi
}


# Render the files to install the given example.
#   render_install_example <example_name> <test_step>
#
# The URLs from the examples will be replaced with the ones used by the tests.
#
# Example:
#   render_install_example "simplest" "00"
# Generates the `00-install.yaml` and `00-assert.yaml` files. Production Jaeger
# will be installed. Its name will be `my-jaeger`.
function render_install_example() {
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for render_install_example. Usage: render_install_example <example_name> <test_step>"
        exit 1
    fi

    example_name=$1
    test_step=$2

    install_file=./$test_step-install.yaml

    # Call `gomplate` instead of `cp` and `mv` to change the name in a single step
    $GOMPLATE -f $EXAMPLES_DIR/$example_name.yaml -o $install_file

    # Fix Elasticsearch URL
    sed -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: $ELASTICSEARCH_URL$ELASTICSEARCH_PORT~gi" $install_file

    # Fix Cassandra URL
    sed -i "s~cassandra.default.svc~$CASSANDRA_SERVER~gi" $install_file

    export JAEGER_NAME
    JAEGER_NAME=$(get_jaeger_name $install_file)
    local jaeger_strategy
    jaeger_strategy=$(get_jaeger_strategy $install_file)

    if [ $jaeger_strategy = "DaemonSet" ] || [ $jaeger_strategy = "allInOne" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ $jaeger_strategy = "production" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/production-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    elif [ $jaeger_strategy = "streaming" ]; then
        $GOMPLATE -f $TEMPLATES_DIR/streaming-jaeger-assert.yaml.template -o ./$test_step-assert.yaml
    else
        error "render_install_example: No strategy declared in the example $example_name. Impossible to determine the assert file to use"
        exit 1
    fi
}


# Render a smoke test for an example.
#   render_smoke_test_example <example_name> <test_step>
#
# Example:
#   render_smoke_test_example "simplest" "01"
# Generates the `01-smoke-test.yaml` and `01-assert.yaml` files.
function render_smoke_test_example() {
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for render_smoke_test_example. Usage: render_smoke_test_example <example_name> <test_step>"
        exit 1
    fi

    example_name=$1
    test_step=$2

    deployment_file=$EXAMPLES_DIR/$example_name.yaml

    jaeger_name=$(get_jaeger_name $deployment_file)
    is_secured="false"

    if [ "$IS_OPENSHIFT" = true  ]; then
        is_secured="true"
    fi

    render_smoke_test "$jaeger_name" "$is_secured" "$test_step"
}



# Render a the Kafka Operator installation
#   render_install_kafka_operator <test_step>
#
# Example:
#   render_install_kafka_operator "01"
# Generates the `01-install.yaml` and `01-assert.yaml` files to install the Kafka
# operator and ensure it is deployed properly.
# Note: the Kafka Operator will not be installed if KAFKA_OLM is `true`.
function render_install_kafka_operator(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for render_install_kafka_operator. Usage: render_install_kafka_operator <test_step>"
        exit 1
    fi

    test_step=$1

    if [ $KAFKA_OLM != true ]; then
        $GOMPLATE -f $TEMPLATES_DIR/kafka-operator-install.yaml.template -o ./$test_step-install.yaml
        $GOMPLATE -f $TEMPLATES_DIR/kafka-operator-assert.yaml.template -o ./$test_step-assert.yaml
    fi
}


# Render a Kafka installation and the files associated to assert all the components.
#   render_install_kafka <cluster_name> <test_step>
#
# Note: 3 assert files are generated, whose names will be $test-step, $test-step+1
# and $test-step+2. If only one file is generated for the 3 deployments to assert,
# the timeout specified in `kuttl-test.yaml` will be used. If each assert is a
# different file, we have 3 * timeout.
#
# Example:
#   render_install_kafka "my-cluster" "01"
# Generates the `01-install.yaml`, `01-assert.yaml`, `02-assert.yaml` and
# `03-assert.yaml` files, installing Kafka and asserting each one of its components.
function render_install_kafka() {
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for render_install_kafka. Usage: render_install_kafka <cluster_name> <test_step>"
        exit 1
    fi

    cluster_name=$1
    test_step=$2

    CLUSTER_NAME=$cluster_name $GOMPLATE -f $TEMPLATES_DIR/kafka-install.yaml.template -o ./$test_step-install.yaml

    render_assert_kafka "false" "$cluster_name" "$test_step"
}

# Render the Kafka cluster assertion files
#   render_install_kafka <autoprovisioned> <cluster_name> <test_step>
#
# Note: 3 assert files are generated, whose names will be $test-step, $test-step+1
# and $test-step+2. If only one file is generated for the 3 deployments to assert,
# the timeout specified in `kuttl-test.yaml` will be used. If each assert is a
# different file, we have 3 * timeout.
#
# Example:
#   render_install_kafka "false" "my-cluster" "01"
# Generates the `01-install.yaml`, `01-assert.yaml`, `02-assert.yaml` and
# `03-assert.yaml` files, installing Kafka and asserting each one of its components.
# The Kafka instance is not autogenerated.
function render_assert_kafka(){
    if [ "$#" -ne 3 ]; then
        error "Wrong number of parameters used for render_assert_kafka. Usage: render_assert_kafka <autoprovisioned> <cluster_name> <test_step>"
        exit 1
    fi

    autoprovisioned=$1
    cluster_name=$2
    test_step=$3

    # This configuration should only be enabled for testing. When it is enabled,
    # the Jaeger Operator will ask the Kafka Operator to deploy only 1 replica
    # of the Kafka cluster and Zookeeper
    if [ "$autoprovisioned" = "true" ] && is_kafka_minimal_enabled; then
        replicas=1
    elif [ "$autoprovisioned" = "true" ]; then
        replicas=3
    elif [ "$autoprovisioned" = "false" ]; then
        replicas=1
    fi

    CLUSTER_NAME=$cluster_name \
        REPLICAS=$replicas \
        $GOMPLATE \
        -f $TEMPLATES_DIR/assert-zookeeper-cluster.yaml.template \
        -o ./$test_step-assert.yaml
    CLUSTER_NAME=$cluster_name \
        REPLICAS=$replicas \
        $GOMPLATE \
        -f $TEMPLATES_DIR/assert-kafka-cluster.yaml.template \
        -o ./$(expr $test_step + 1 )-assert.yaml
    CLUSTER_NAME=$cluster_name \
        $GOMPLATE \
        -f $TEMPLATES_DIR/assert-entity-operator.yaml.template \
        -o ./0$(expr $test_step + 2 )-assert.yaml
}


# Render a "find service" job.
#   render_find_service <jaeger_name> <deployment_strategy> <service_name> <job_number> <test_step>
#
# Example:
#   render_find_service "simplest" "production" "my-service" "01" "00"
# Generates the `01-find-service.yaml` and `01-assert.yaml` files. It will run a
# `find-service` job.
# Accepted values for <deployment_strategy>:
#   * allInOne: all in one deployment.
#   * production: production using Elasticsearch.
function render_find_service() {
    if [ "$#" -ne 5 ]; then
        error "Wrong number of parameters used for render_find_service. Usage: render_find_service <jaeger_name> <deployment_strategy> <service_name> <job_number> <test_step>"
        exit 1
    fi

    jaeger=$1
    deployment_strategy=$2
    service_name=$3
    job_number=$4
    test_step=$5

    export JAEGER_NAME=$jaeger
    export JOB_NUMBER=$job_number
    export SERVICE_NAME=$service_name
    export JAEGER_QUERY_ENDPOINT

    if [ $IS_OPENSHIFT = true ] && [ $deployment_strategy != "allInOne" ]; then
        protocol="https://"
        query_port=""
        template="$TEMPLATES_DIR/openshift/find-service.yaml.template"
        JAEGER_QUERY_ENDPOINT="https://$jaeger-query"
    else
        template="$TEMPLATES_DIR/find-service.yaml.template"
        JAEGER_QUERY_ENDPOINT="http://$jaeger-query:16686"
    fi


    $GOMPLATE -f $template -o ./$test_step-find-service.yaml
    $GOMPLATE -f $TEMPLATES_DIR/assert-find-service.yaml.template -o ./$test_step-assert.yaml

    unset JAEGER_NAME
    unset SERVICE_NAME
    unset JOB_NUMBER
    unset JAEGER_COLLECTOR_ENDPOINT
}


# Render a tracegen deployment.
#   render_install_tracegen <jaeger_name> <test_step>
#
# Example:
#   render_install_tracegen "prod" "00"
# Generates the `00-install.yaml` and `00-assert.yaml` files. It will deploy
# 1 replica of the tracegen deployment.
function render_install_tracegen() {
    if [ "$#" -ne 2 ]; then
        error "Wrong number of parameters used for render_install_tracegen. Usage: render_install_tracegen <jaeger_name> <test_step>"
        exit 1
    fi

    jaeger=$1
    step=$2

    # We detected this value is good enough to make the operator scale
    replicas=1

    $GOMPLATE -f $EXAMPLES_DIR/tracegen.yaml -o ./$step-install.yaml
    $YQ e -i ".spec.replicas=$replicas" ./$step-install.yaml
    sed -i "s~simple-prod~$jaeger~gi" ./$step-install.yaml
    REPLICAS=$replicas $GOMPLATE -f $TEMPLATES_DIR/assert-tracegen.yaml.template -o ./$step-assert.yaml
}

# Get the Jaeger name from a Jaeger deployment file.
#   get_jaeger_name <file>
#
# Example:
#   get_jaeger_name "my-jaeger-deployment.yaml"
# Returns the name of the Jaeger deployment.
function get_jaeger_name() {
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for get_jaeger_name. Usage: get_jaeger_name <file>"
        exit 1
    fi

    deployment_file=$1

    jaeger_name=$($YQ e '. | select(.kind == "Jaeger").metadata.name' $deployment_file)

    if [ -z "$jaeger_name" ]; then
        error "No name for Jaeger deployment in file $deployment_file"
        cat $deployment_file
        exit 1
    fi

    echo $jaeger_name
    return 0
}


# Get the Jaeger deployment strategy from a Jaeger deployment file.
#   get_jaeger_strategy <file>
#
# Example:
#   get_jaeger_strategy "my-jaeger-deployment.yaml"
# Returns the name of the Jaeger deployment.
function get_jaeger_strategy() {
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for get_jaeger_strategy. Usage: get_jaeger_strategy <file>"
        exit 1
    fi

    deployment_file=$1

    strategy=$($YQ e '. | select(.kind == "Jaeger").spec.strategy' $deployment_file)

    if [ "$strategy" = "production" ] || [ "$strategy" = "streaming" ]; then
        echo $strategy
        return 0
    fi

    strategy=$($YQ e '. | select(.kind == "Jaeger").spec.agent.strategy' $deployment_file)
    if [ "$strategy" = "null" ]; then
        echo "allInOne"
        return 0
    fi

    echo $strategy
    return 0
}



# Start a test. It will move to the correct folder and prin an header.
#   start_test <test_name>
#
# Example:
#   start_test "my_test"
# Returns the name of the Jaeger deployment.
function start_test() {
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for start_test. Usage: start_test <test_name>"
        exit 1
    fi

    test_name=$1

    echo "==========================================================================="
    info "Rendering files for test $test_name"
    echo "==========================================================================="

    if [ "$(basename `pwd`)" != "_build" ]; then
        cd ..
    fi

    mkdir -p $test_name
    cd $test_name
}


# Render the files to install and assert a Vertx deploymnet.
#    render_install_vertx <test_step>
#
# Example:
#   render_install_vertx "01"
# This call will deploy a vertex instance as the step 01.
function render_install_vertx(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for render_install_vertx. Usage: render_install_vertx <test_step>"
        exit 1
    fi

    test_step=$1

    $GOMPLATE -f $TEMPLATES_DIR/vertex-install.yaml.template -o ./$test_step-install.yaml
    $GOMPLATE -f $TEMPLATES_DIR/vertex-assert.yaml.template -o ./$test_step-assert.yaml
}


# Print an info message
#   message <message>
#
# Example:
#   info "Something is happening"
function info(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for error. Usage: info <message>"
        exit 1
    fi

    echo -e "\e[1;34m$1\e[0m"
}


# Print an error
#   error <message>
#
# Example:
#   error "Something is happening"
function error(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for error. Usage: error <message>"
        exit 1
    fi

    echo -e "\e[1;31mERR: $1\e[0m"
}


# Print a warning
#   warning <message>
#
# Example:
#   warning "Something is happening"
function warning(){
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for warning. Usage: warning <message>"
        exit 1
    fi

    echo -e "\e[1;33mWAR: $1\e[0m"
}

# Scape command. The result is returned in the CMD_PARAMETERS environment
# variable.
#    escape_command <command_to_escape>
#
# Example:
#   escape_command "'--pattern', 'jaeger-span-\d{4}-\d{2}-\d{2}',"
function escape_command(){
    if [ "$#" -ne 1 ]; then
        error "Missing argument for escape_command. Usage: escape_command <command>"
        exit 1
    fi

    command=$1

    export CMD_PARAMETERS=$(echo "$command" | sed 's/\\/\\\\/g')
}


# Skip test. The test is skipped.
#    skip_test <test_name> <message>
#
# Example:
#   skip_test "elasticsearch-simple" "The test is not supported"
function skip_test(){
    if [ "$#" -ne 2 ]; then
        error "Missing argument for skip_test. Usage: skip_test <test_name> <message>"
        exit 1
    fi

    test_name=$1
    message=$2

    if [ "$(basename `pwd`)" != "_build" ]; then
        cd ..
    fi

    # Remove the folder of the test because some files could be copied
    rm -rf $test_name

    warning "$test_name: $message"
}

# Get the installed version of Elasticsearch OpenShift Operator in the cluster.
#   get_elasticsearch_openshift_operator_version
#
# Notes:
#   - It throws an error if it is used in a non OpenShift cluster.
#   - It throws an error if the Elasticsearch OpenShift Operator is not installed.
#   - The result is returned in the ESO_OPERATOR_VERSION shell variable.
#
function get_elasticsearch_openshift_operator_version(){
    export ESO_OPERATOR_VERSION
    if [ "$IS_OPENSHIFT" = true ]; then
        properties=$(kubectl get pods -l name=elasticsearch-operator --all-namespaces -o=jsonpath='{.items[0].metadata.annotations.operatorframework\.io/properties}')
        if [ -z "$properties" ]; then
            error "Elasticsearch OpenShift Operator not found"
            exit 1
        fi
        ESO_OPERATOR_VERSION=$(echo "$properties" | $YQ e -P '.properties.[] | select(.value.packageName == "elasticsearch-operator") | .value.version')
    else
        error "Not an OpenShift cluster. Impossible to get the Elasticsearch OpenShift Operator version"
        exit 1
    fi
}

function version_gt() {
    test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" != "$1";
}

function version_ge() {
    test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" == "$1";
}

function version_le(){
    test "$(echo "$@" | tr " " "\n" | sort -V | head -n 1)" == "$1";
}
function version_lt() {
    test "$(echo "$@" | tr " " "\n" | sort -rV | head -n 1)" != "$1";
}


# Check if the KAFKA-PROVISION-MINIMAL feature is enabled.
function is_kafka_minimal_enabled() {
    namespaces=( observability openshift-operators openshift-distributed-tracing )
    for i in "${namespaces[@]}"
    do
        enabled="$(kubectl get pods -n $i -l name=jaeger-operator -o yaml | $YQ e '.items[0].spec.containers[0].env[] | select(.name=="KAFKA-PROVISIONING-MINIMAL").value')"
        if [ "$enabled" == true ]; then
            return 0
        fi
    done
    return 1
}



###############################################################################
# Init configuration ##########################################################
###############################################################################

# Enable verbosity
if [ "$VERBOSE" = true ]; then
    set -o xtrace
fi


# Check if we are using an OpenShift cluster or not
output=$(kubectl get clusterversion  2> /dev/null)
IS_OPENSHIFT=false
if [ ! -z "$output" ]; then
    warning "Generating templates for an OpenShift cluster"
    IS_OPENSHIFT=true
fi

export KAFKA_USE_CUSTOM_PODSET
if [ -z "$KAFKA_VERSION" ] || version_le $KAFKA_VERSION "0.25.0" ; then
    KAFKA_USE_CUSTOM_PODSET=false
else
    KAFKA_USE_CUSTOM_PODSET=true
fi

export IS_OPENSHIFT

# Important folders
export SUITE_DIR=$(dirname "$0")


# Check the dependencies are there
$ROOT_DIR/hack/install/install-gomplate.sh
$ROOT_DIR/hack/install/install-yq.sh
$ROOT_DIR/hack/install/install-kustomize.sh


# Elasticsearch settings
export ELASTICSEARCH_NODECOUNT="1"
export ELASTICSEARCH_URL="http://elasticsearch"
export ELASTICSEARCH_PORT=":9200"


# Cassandra settings
export CASSANDRA_SERVER="cassandra"

export SERVICE_ACCOUNT_NAME="e2e-test"


# Programs. Note: these paths are related to the suites location
PROGRAMS_FOLDER=../../../..

# CMD utils
export ASSERT_HTTP_CODE_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/assert-jaeger-http-code.sh
export CHECK_JAEGER_VERSION_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/check-jaeger-version.sh
export CHECK_JAEGER_OPERATOR_VERSION_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/check-jaeger-operator-version.sh
export GET_TOKEN_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/get-token.sh
export TEST_UI_CONFIG_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/uiconfig/main.go
export WAIT_CRONJOB_PROGRAM=$PROGRAMS_FOLDER/cmd-utils/wait-cronjob/main.go

# Assert jobs
export QUERY_PROGRAM=$PROGRAMS_FOLDER/assert-jobs/query/main.go
export REPORTER_PROGRAM=$PROGRAMS_FOLDER/assert-jobs/reporter/main.go


# Fail on first error
set -e

# Move to the suite folder
cd $(pwd)/$SUITE_DIR

# Clean the previously generated templates and copy all the non template files
# to the _build folder
build_dir="_build"
rm -rf $build_dir
mkdir $build_dir

find -maxdepth 1 -type d ! -wholename "." ! -wholename "./$build_dir" | xargs -I {} cp -r {}  $build_dir

cd _build

info "Rendering kuttl-test.yaml"

if [ $IS_OPENSHIFT = true ]; then
    CRD_DIR=""
else
    CRD_DIR="../../../_build/crds/"
fi
export CRD_DIR

$GOMPLATE -f ../../../templates/kuttl-test.yaml.template -o ./kuttl-test.yaml
mkdir -p artifacts
