#!/bin/bash

go version

echo "Installing kubectl"
curl -sLo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.10.0/bin/linux/amd64/kubectl > /dev/null
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

echo "Installing minikube"
curl -sLo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 > /dev/null
chmod +x minikube 
sudo mv minikube /usr/local/bin/

echo "Installing go dep"
curl -sLo dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 > /dev/null
chmod +x dep
sudo mv dep /usr/local/bin/

echo "Installing gosec"
go get github.com/securego/gosec/cmd/gosec/...

echo "Installing golint"
go get -u golang.org/x/lint/golint

echo "Installing the operator-sdk command"
mkdir -p $GOPATH/src/github.com/operator-framework
cd $GOPATH/src/github.com/operator-framework
git clone https://github.com/operator-framework/operator-sdk > /dev/null
cd operator-sdk
git checkout master > /dev/null
make dep > /dev/null
make install > /dev/null
cd ${TRAVIS_BUILD_DIR}

echo "Starting a Kubernetes cluster with minikube/localkube"
sudo minikube start --vm-driver=none --kubernetes-version=v1.10.0 --bootstrapper=localkube > /dev/null

echo "Updating minikube context"
minikube update-context > /dev/null

echo "Waiting for the Kubernetes cluster to get ready"
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

echo "Performing a 'docker login' operation"
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

echo "Initializing an Elasticsearch cluster"
make es
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get pods -lapp=jaeger-elasticsearch -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
