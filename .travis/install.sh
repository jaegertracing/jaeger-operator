#!/bin/bash

echo "Installing gosec"
go get github.com/securego/gosec/cmd/gosec/...

echo "Installing golint"
go get -u golang.org/x/lint/golint
