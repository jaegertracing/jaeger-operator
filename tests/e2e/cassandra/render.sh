#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "cassandra-smoke"
export jaeger_name=with-cassandra

# Create Cassandra instance and assert it
render_install_cassandra "00"

# Create the Jaeger instance
render_install_jaeger "$jaeger_name" "production_cassandra" "01"

# Run smoke test
render_smoke_test "$jaeger_name" "production" "02"


start_test "cassandra-spark"
# Create Cassandra instance and assert it
render_install_cassandra "00"

# Create the Jaeger instance
export JAEGER_NAME=test-spark-deps
export DEP_SCHEDULE=true
export CASSANDRA_MODE=prod
$GOMPLATE -f $TEMPLATES_DIR/cassandra-jaeger-install.yaml.template -o ./01-install.yaml
