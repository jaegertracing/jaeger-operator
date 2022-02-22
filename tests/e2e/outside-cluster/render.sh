#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "outside-cluster"
jaeger_name="my-jaeger"
render_install_elasticsearch "00"
render_install_jaeger "$jaeger_name" "production" "01"
$GOMPLATE -f ./03-check-collector.yaml.template -o 03-check-collector.yaml
