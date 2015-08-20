# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth evm mist all test travis-test-with-coverage clean
GOBIN = build/bin

geth:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

evm:
	build/env.sh $(GOROOT)/bin/go install -v $(shell build/ldflags.sh) ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm to start the evm."

rlpdump:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/rlpdump
	@echo "Done building."
	@echo "Run \"$(GOBIN)/rlpdump\" to launch rlpdump."

disasm:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/disasm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/disasm\" to launch disasm."

ethtest:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/ethtest
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ethtest\" to launch ethtest."

bootnode:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/bootnode
	@echo "Done building."
	@echo "Run \"$(GOBIN)/bootnode\" to launch bootnode."

all:
	build/env.sh go install -v $(shell build/ldflags.sh) ./...

test: all
	build/env.sh go test ./...

travis-test-with-coverage: all
	build/env.sh build/test-global-coverage.sh

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
