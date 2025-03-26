#!/bin/bash

# Install protobuf
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    os="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    os="osx"
else
    echo "Unsupported platform"
    exit 1
fi

PROTOC_ZIP=protoc-3.19.3-$os-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.19.3/$PROTOC_ZIP
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP

# Change permissions to use the binary
sudo chmod -R 755 /usr/local/bin/protoc
sudo chmod -R 755 /usr/local/include

# Install golang extensions (DO NOT CHANGE THE VERSIONS)
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
