#!/bin/bash

echo "Installing gosec"
go get github.com/securego/gosec/cmd/gosec/...
pushd
cd $GOPATH/src/github.com/securego/gosec
git checkout 1.3.0
rm $GOPATH/bin/gosec
go install cmd/gosec
popd

echo "Installing golint"
go get -u golang.org/x/lint/golint

echo "Installing operator-sdk"
curl https://github.com/operator-framework/operator-sdk/releases/download/v0.5.0/operator-sdk-v0.5.0-x86_64-linux-gnu -sLo $GOPATH/bin/operator-sdk
chmod +x $GOPATH/bin/operator-sdk
