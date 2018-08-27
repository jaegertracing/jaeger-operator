#!/bin/bash

echo 'Build succeeded, operator was generated, Jaeger operator is running on minikube, and unit/integration tests pass'

echo "Uploading code coverage results"
bash <(curl -s https://codecov.io/bash)