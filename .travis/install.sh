#!/bin/bash

# echo "Installing kubectl"
# curl -sLo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.10.0/bin/linux/amd64/kubectl > /dev/null
# chmod +x kubectl
# sudo mv kubectl /usr/local/bin/

echo "Installing gosec"
go get github.com/securego/gosec/cmd/gosec/...

echo "Installing golint"
go get -u golang.org/x/lint/golint

# if [ ! -d ${HOME}/google-cloud-sdk ]; then
#     curl https://sdk.cloud.google.com | bash;
# fi

# gcloud auth activate-service-account --key-file test/client-secret.json
# gcloud container clusters get-credentials jpkroehling-jaeger-operator-master --zone us-central1-a --project jpkroehling-jaeger-operator
