# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth geth-cross evm all test travis-test-with-coverage xgo clean
.PHONY: geth-linux geth-linux-arm geth-linux-386 geth-linux-amd64
.PHONY: geth-darwin geth-darwin-386 geth-darwin-amd64
.PHONY: geth-windows geth-windows-386 geth-windows-amd64
.PHONY: geth-android geth-android-16 geth-android-21

GOBIN = build/bin

CROSSDEPS = https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2
GO ?= latest

geth:
	build/env.sh go install -v $(shell build/flags.sh) ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

geth-cross: geth-linux geth-darwin geth-windows geth-android
	@echo "Full cross compilation done:"
	@ls -l $(GOBIN)/geth-*

geth-linux: xgo geth-linux-arm geth-linux-386 geth-linux-amd64
	@echo "Linux cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-*

geth-linux-arm: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/arm -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux ARM cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm

geth-linux-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep 386

geth-linux-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep amd64

geth-darwin: xgo geth-darwin-386 geth-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-*

geth-darwin-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=darwin/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Darwin 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-* | grep 386

geth-darwin-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=darwin/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Darwin amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-* | grep amd64

geth-windows: xgo geth-windows-386 geth-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-*

geth-windows-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=windows/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Windows 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-* | grep 386

geth-windows-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=windows/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Windows amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-* | grep amd64

geth-android: xgo geth-android-16 geth-android-21
	@echo "Android cross compilation done:"
	@ls -l $(GOBIN)/geth-android-*

geth-android-16: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=android-16/* -v $(shell build/flags.sh) ./cmd/geth
	@echo "Android 16 cross compilation done:"
	@ls -l $(GOBIN)/geth-android-16-*

geth-android-21: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=android-21/* -v $(shell build/flags.sh) ./cmd/geth
	@echo "Android 21 cross compilation done:"
	@ls -l $(GOBIN)/geth-android-21-*

evm:
	build/env.sh $(GOROOT)/bin/go install -v $(shell build/flags.sh) ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm to start the evm."

all:
	build/env.sh go install -v $(shell build/flags.sh) ./...

test: all
	build/env.sh go test ./...

travis-test-with-coverage: all
	build/env.sh build/test-global-coverage.sh

xgo:
	build/env.sh go get github.com/karalabe/xgo

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
