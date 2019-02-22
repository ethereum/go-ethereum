.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    go-ethereum-timing/cmd/swarm
	go build -o bin/geth     go-ethereum-timing/cmd/geth
	go build -o bin/bootnode go-ethereum-timing/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v go-ethereum-timing/swarm/...
