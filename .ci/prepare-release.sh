#!/bin/bash

if [[ -z $OPERATOR_VERSION ]]; then
    echo "OPERATOR_VERSION isn't set. Skipping process."
    exit 1
fi



JAEGER_VERSION=$(echo $JAEGER_VERSION | tr -d '"')
JAEGER_AGENT_VERSION=$(echo $JAEGER_AGENT_VERSION | tr -d '"')


PREVIOUS_VERSION=$(grep operator= versions.txt | awk -F= '{print $2}')

# change the versions.txt, bump only operator version.
sed "s~operator=${PREVIOUS_VERSION}~operator=${OPERATOR_VERSION}~gi" -i versions.txt

# changes to deploy/operator.yaml
sed "s~replaces: jaeger-operator.v.*~replaces: jaeger-operator.v${PREVIOUS_VERSION}~i" -i config/manifests/bases/jaeger-operator.clusterserviceversion.yaml

# Update the examples according to the release

sed -i "s~all-in-one:.*~all-in-one:${JAEGER_VERSION}~gi" examples/all-in-one-with-options.yaml

# statefulset-manual-sidecar
sed -i "s~jaeger-agent:.*~jaeger-agent:${JAEGER_AGENT_VERSION}~gi" examples/statefulset-manual-sidecar.yaml

# operator-with-tracing
sed -i "s~jaeger-operator:.*~jaeger-operator:${OPERATOR_VERSION}~gi" examples/operator-with-tracing.yaml
sed -i "s~jaeger-agent:.*~jaeger-agent:${JAEGER_AGENT_VERSION}~gi" examples/operator-with-tracing.yaml

# tracegen
sed -i "s~jaeger-tracegen:.*~jaeger-tracegen:${JAEGER_VERSION}~gi" examples/tracegen.yaml


VERSION=${OPERATOR_VERSION} USER=jaegertracing make bundle
