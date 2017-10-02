.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/burnout/go-burnout/cmd/swarm
	go build -o bin/geth     github.com/burnout/go-burnout/cmd/geth
	go build -o bin/bootnode github.com/burnout/go-burnout/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/burnout/go-burnout/swarm/...
