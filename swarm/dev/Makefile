.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/burnoutcoin/go-burnout/cmd/swarm
	go build -o bin/geth     github.com/burnoutcoin/go-burnout/cmd/geth
	go build -o bin/bootnode github.com/burnoutcoin/go-burnout/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/burnoutcoin/go-burnout/swarm/...
