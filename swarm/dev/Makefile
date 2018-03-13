.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/TeamEGEM/go-egem/cmd/swarm
	go build -o bin/geth     github.com/TeamEGEM/go-egem/cmd/geth
	go build -o bin/bootnode github.com/TeamEGEM/go-egem/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/TeamEGEM/go-egem/swarm/...
