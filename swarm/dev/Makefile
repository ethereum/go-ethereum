.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/expanse-org/go-expanse/cmd/swarm
	go build -o bin/geth     github.com/expanse-org/go-expanse/cmd/geth
	go build -o bin/bootnode github.com/expanse-org/go-expanse/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/expanse-org/go-expanse/swarm/...
