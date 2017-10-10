.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/EthereumCommonwealth/go-callisto/cmd/swarm
	go build -o bin/geth     github.com/EthereumCommonwealth/go-callisto/cmd/geth
	go build -o bin/bootnode github.com/EthereumCommonwealth/go-callisto/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/EthereumCommonwealth/go-callisto/swarm/...
