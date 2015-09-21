# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth evm mist all test travis-test-with-coverage clean
GOBIN = build/bin

geth:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

geth-cross: geth-linux geth-darwin geth-windows geth-android
	@echo "Full cross compilation done:"
	@ls -l $(GOBIN)/geth-*

geth-linux: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=linux/* -v ./cmd/geth
	@echo "Linux cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-*

geth-darwin: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=darwin/* -v ./cmd/geth
	@echo "Darwin cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-*

geth-windows: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=windows/* -v ./cmd/geth
	@echo "Windows cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-*

geth-android: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=android-16/*,android-21/* -v ./cmd/geth
	@echo "Android cross compilation done:"
	@ls -l $(GOBIN)/geth-android-*

evm:
	build/env.sh $(GOROOT)/bin/go install -v $(shell build/ldflags.sh) ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm to start the evm."
mist:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/mist
	@echo "Done building."
	@echo "Run \"$(GOBIN)/mist --asset_path=cmd/mist/assets\" to launch mist."

all:
	build/env.sh go install -v $(shell build/ldflags.sh) ./...

test: all
	build/env.sh go test ./...

travis-test-with-coverage: all
	build/env.sh build/test-global-coverage.sh

xgo:
	build/env.sh go get github.com/karalabe/xgo

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
