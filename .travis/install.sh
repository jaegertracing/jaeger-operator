#!/bin/bash

echo "Installing gosec"
go get github.com/securego/gosec/cmd/gosec/...

echo "Installing golint"
go get -u golang.org/x/lint/golint

echo "Installing operator-sdk"
curl https://github.com/operator-framework/operator-sdk/releases/download/v0.0.7/operator-sdk-v0.0.7-x86_64-linux-gnu -sLo $GOPATH/bin/operator-sdk
chmod +x $GOPATH/bin/operator-sdk