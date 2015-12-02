# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth geth-cross evm all test travis-test-with-coverage xgo clean
.PHONY: geth-linux geth-linux-386 geth-linux-amd64
.PHONY: geth-linux-arm geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
.PHONY: geth-darwin geth-darwin-386 geth-darwin-amd64
.PHONY: geth-windows geth-windows-386 geth-windows-amd64
.PHONY: geth-android geth-ios

GOBIN = build/bin

CROSSDEPS = https://gmplib.org/download/gmp/gmp-6.1.0.tar.bz2
MODE ?= default
GO ?= latest

geth:
	build/env.sh go install -v $(shell build/flags.sh) ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

geth-cross: geth-linux geth-darwin geth-windows geth-android geth-ios
	@echo "Full cross compilation done:"
	@ls -l $(GOBIN)/geth-*

geth-linux: geth-linux-386 geth-linux-amd64 geth-linux-arm
	@echo "Linux cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-*

geth-linux-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep 386

geth-linux-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep amd64

geth-linux-arm: geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm

geth-linux-arm-5: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/arm-5 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux ARMv5 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm-5

geth-linux-arm-6: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/arm-6 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux ARMv6 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm-6

geth-linux-arm-7: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/arm-7 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux ARMv7 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm-7

geth-linux-arm64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=linux/arm64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Linux ARM64 cross compilation done:"
	@ls -l $(GOBIN)/geth-linux-* | grep arm64

geth-darwin: geth-darwin-386 geth-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-*

geth-darwin-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=darwin/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Darwin 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-* | grep 386

geth-darwin-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=darwin/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Darwin amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-darwin-* | grep amd64

geth-windows: geth-windows-386 geth-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-*

geth-windows-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=windows/386 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Windows 386 cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-* | grep 386

geth-windows-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=windows/amd64 -v $(shell build/flags.sh) ./cmd/geth
	@echo "Windows amd64 cross compilation done:"
	@ls -l $(GOBIN)/geth-windows-* | grep amd64

geth-android: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --targets=android/* -v $(shell build/flags.sh) ./cmd/geth
	@echo "Android cross compilation done:"
	@ls -l $(GOBIN)/geth-android-*

geth-ios: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --buildmode=$(MODE) --dest=$(GOBIN) --deps=$(CROSSDEPS) --depsargs=--disable-assembly --targets=ios/* -v $(shell build/flags.sh) ./cmd/geth
	@echo "iOS cross compilation done:"
	@ls -l $(GOBIN)/geth-ios-*

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
