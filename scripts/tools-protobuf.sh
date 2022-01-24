#!/bin/bash

# Install protobuf
PROTOC_ZIP=protoc-3.12.0-linux-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.12.0/$PROTOC_ZIP
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP

# Change permissions to use the binary
sudo chmod 755 -R /usr/local/bin/protoc
sudo chmod 755 -R /usr/local/include

# Install golang extensions (DO NOT CHANGE THE VERSIONS)
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
