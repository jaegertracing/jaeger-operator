#!/bin/bash
VERSION=3.4.20

echo "Installing etcd"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="etcd"

create_bin

check_tool "$BIN/$PROGRAM" $VERSION "--version"

url="https://github.com/etcd-io/etcd/releases/download/v${VERSION}/etcd-v${VERSION}-linux-amd64.tar.gz -o /tmp/etcd-v${VERSION}-linux-amd64.tar.gz"

retry "curl -L $url -o /tmp/etcd-v${VERSION}-linux-amd64.tar.gz"

mkdir /tmp/etcd-download-test
tar xzvf /tmp/etcd-v${VERSION}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1
rm -f /tmp/etcd-v${VERSION}-linux-amd64.tar.gz

mv /tmp/etcd-download-test/etcd $BIN
mv /tmp/etcd-download-test/etcdctl $BIN
