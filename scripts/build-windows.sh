#!/usr/bin/env bash

set -eo pipefail

rm -f rsrc_windows_*
rm -f habo-client.exe
rm -f habo-client.*.bak
rm -f .habo-client.*.old

rm -f habo-client-amd64-installer.exe

sudo apt-get update && sudo apt-get install -y nsis

go install github.com/tc-hib/go-winres@v0.3.1

export PATH="$PATH:$(go env GOPATH)/bin"

go-winres make

(cd frontend && npm ci && npm run build)
env GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui -X main.version=$GITHUB_REF_NAME" -o habo-client.exe albiondata-client.go

go-winres patch habo-client.exe

cd pkg/nsis
make nsis

cd ../..
ls -la habo-client*

cp habo-client.exe habo-client.exe.copy
gzip -9 habo-client.exe
mv habo-client.exe.gz update-windows-amd64.exe.gz
mv habo-client.exe.copy habo-client.exe
