#!/usr/bin/env bash

set -eoux pipefail

export OPENSHIFT_VERSION=v3.9
export KUBERNETES_VERSION=v1.14.0

# setup insecure registry
echo '{"registry-mirrors": ["https://mirror.gcr.io"], "mtu": 1460, "insecure-registries": ["172.30.0.0/16"] }' | sudo tee /etc/docker/daemon.json
sudo service docker restart

# install and run OCP
sudo docker cp $(docker create docker.io/openshift/origin:$OPENSHIFT_VERSION):/bin/oc /usr/local/bin/oc
oc cluster up --version=$OPENSHIFT_VERSION
oc login -u system:admin

# download kubectl
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl &&  \
    sudo mv kubectl /usr/local/bin/

