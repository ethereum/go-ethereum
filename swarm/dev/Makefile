.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/XinFinOrg/XDPoSChain/cmd/swarm
	go build -o bin/XDC     github.com/XinFinOrg/XDPoSChain/cmd/XDC
	go build -o bin/bootnode github.com/XinFinOrg/XDPoSChain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/XinFinOrg/XDPoSChain/swarm/...
