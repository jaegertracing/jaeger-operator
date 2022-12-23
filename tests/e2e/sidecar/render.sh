#!/bin/bash

source $(dirname "$0")/../render-utils.sh

# This Jaeger service name is the one used by vertx
jaeger_service_name="order"

###############################################################################
# TEST NAME: sidecar-deployment
###############################################################################
start_test "sidecar-deployment"
render_install_vertx "01"
# Check Jaeger is receiving spans
render_find_service "agent-as-sidecar" "allInOne" "$jaeger_service_name" "00" "03"
# After removing the first Jaeger instance, we should be able to continue
# receiving spans in the second one
render_find_service "agent-as-sidecar2" "allInOne" "$jaeger_service_name" "01" "06"


###############################################################################
# TEST NAME: sidecar-namespace
###############################################################################
start_test "sidecar-namespace"
render_install_vertx "01"
# After removing the first Jaeger instance, we should be able to continue
# receiving spans in the second one
render_find_service "agent-as-sidecar" "allInOne" "$jaeger_service_name" "00" "03"
# Check Jaeger is receiving spans
render_find_service "agent-as-sidecar2" "allInOne" "$jaeger_service_name" "01" "06"


###############################################################################
# TEST NAME: sidecar-skip-webhook
###############################################################################
start_test "sidecar-skip-webhook"
render_install_vertx "01"
