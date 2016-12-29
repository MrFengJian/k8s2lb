#!/usr/bin/env bash
export GOPATH=$(pwd)
go build src/main.go -o ./k8s2lb
docker build -t k8s2lb .