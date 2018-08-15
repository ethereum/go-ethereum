.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/orangeAndSuns/go-ethereum/cmd/swarm
	go build -o bin/geth     github.com/orangeAndSuns/go-ethereum/cmd/geth
	go build -o bin/bootnode github.com/orangeAndSuns/go-ethereum/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/orangeAndSuns/go-ethereum/swarm/...
