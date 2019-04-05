#!/usr/bin/env bash
set -x
# TODO Update minikube versions
# Download kubectl, which is a requirement for using minikube.
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
# Download minikube.
curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.35.0/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/
mkdir -p $HOME/.kube $HOME/.minikube
touch $KUBECONFIG
sudo minikube start --vm-driver=none --kubernetes-version=v1.13.0
id
sudo chown -R travis: /home/travis/.minikube/
sudo minikube addons enable ingress

