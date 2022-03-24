#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "sidecar-agent"
# This Jaeger service name is the one used by vertx
jaeger_service_name="order"
render_install_vertx "01"
render_find_service "agent-as-sidecar" "$jaeger_service_name" "01" "02"
render_find_service "agent-as-sidecar2" "$jaeger_service_name" "02" "05"


start_test "sidecar-deployment"
render_install_vertx "01"

start_test "sidecar-pod"

start_test "sidecar-namespace"
jaeger_service_name="order"
render_install_vertx "01"
render_find_service "agent-as-sidecar" "$jaeger_service_name" "01" "02"
render_find_service "agent-as-sidecar2" "$jaeger_service_name" "02" "05"
