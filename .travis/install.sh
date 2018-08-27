#!/bin/bash

echo "Using image: ${BUILD_IMAGE}"
sed "s~image: jaegertracing\/jaeger-operator\:.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml
echo "Resulting operator.yaml:"
cat deploy/operator.yaml
