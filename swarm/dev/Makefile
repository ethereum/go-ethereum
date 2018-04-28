.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/ethereum/go-etherfact/cmd/swarm
	go build -o bin/getf     github.com/ethereum/go-etherfact/cmd/getf
	go build -o bin/bootnode github.com/ethereum/go-etherfact/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/ethereum/go-etherfact/swarm/...
