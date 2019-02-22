.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/ubiq/go-ubiq/cmd/swarm
	go build -o bin/gubiq     github.com/ubiq/go-ubiq/cmd/gubiq
	go build -o bin/bootnode github.com/ubiq/go-ubiq/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/ubiq/go-ubiq/swarm/...
