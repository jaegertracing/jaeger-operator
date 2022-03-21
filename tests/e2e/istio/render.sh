#!/bin/bash

source $(dirname "$0")/../render-utils.sh

if [ $IS_OPENSHIFT = true ]; then
    skip_test "istio" "Test not supported in OpenShift"
else
    start_test "istio"
    export jaeger_name="simplest"
    cat $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./livelinessprobe.template > ./03-install.yaml
    render_find_service "$jaeger_name" "order" "00" "04"

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
