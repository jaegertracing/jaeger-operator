#!/usr/bin/env bash

set -eoux pipefail

export OPENSHIFT_VERSION=v3.9
export KUBERNETES_VERSION=v1.14.0

# setup insecure registry
echo '{"registry-mirrors": ["https://mirror.gcr.io"], "mtu": 1460, "insecure-registries": ["172.30.0.0/16"] }' | sudo tee /etc/docker/daemon.json
sudo service docker restart

HOST_IP=$(hostname -I | awk '{ print $1 }')

# install and run OCP
sudo docker cp $(docker create docker.io/openshift/origin:$OPENSHIFT_VERSION):/bin/oc /usr/local/bin/oc

oc cluster up --version=$OPENSHIFT_VERSION --public-hostname=${HOST_IP}.nip.io
oc login -u system:admin
