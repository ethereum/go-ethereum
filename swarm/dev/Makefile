.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/eximchain/go-ethereum/cmd/swarm
	go build -o bin/geth     github.com/eximchain/go-ethereum/cmd/geth
	go build -o bin/bootnode github.com/eximchain/go-ethereum/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/eximchain/go-ethereum/swarm/...
