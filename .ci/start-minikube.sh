#!/usr/bin/env bash
# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u
# print each command before executing it
set -x

#
# NOTE: This script was originally copied from the CoreOs Prometheus Operator build
# https://github.com/coreos/prometheus-operator/blob/master/scripts/create-minikube.sh

# socat is needed for port forwarding
sudo apt-get update && sudo apt-get install socat

env | sort

#export MINIKUBE_VERSION=v1.0.0
export KUBERNETES_VERSION=v1.19.2

MINIKUBE=$(which minikube) # it's outside of the regular PATH, so, need the full path when calling with sudo

sudo mount --make-rshared /
sudo mount --make-rshared /proc
sudo mount --make-rshared /sys

mkdir "${HOME}"/.kube || true
touch "${HOME}"/.kube/config

# minikube config
minikube config set WantNoneDriverWarning false
minikube config set vm-driver none

whoami
minikube version
sudo ${MINIKUBE} start --kubernetes-version=$KUBERNETES_VERSION --extra-config=apiserver.authorization-mode=RBAC --driver=none
#sudo chown -R $USER $HOME/.kube $HOME/.minikube
ls -alF ${HOME}

minikube update-context

# waiting for node(s) to be ready
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

# waiting for kube-dns to be ready
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl -n kube-system get pods -lk8s-app=kube-dns -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1;echo "waiting for kube-dns to be available"; kubectl get pods --all-namespaces; done

minikube addons enable ingress

eval $(minikube docker-env)