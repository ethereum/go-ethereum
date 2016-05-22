# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gexp gexp-cross evm all test travis-test-with-coverage xgo clean
.PHONY: gexp-linux gexp-linux-386 gexp-linux-amd64
.PHONY: gexp-linux-arm gexp-linux-arm-5 gexp-linux-arm-6 gexp-linux-arm-7 gexp-linux-arm64
.PHONY: gexp-darwin gexp-darwin-386 gexp-darwin-amd64
.PHONY: gexp-windows gexp-windows-386 gexp-windows-amd64
.PHONY: gexp-android gexp-ios

GOBIN = build/bin
GO ?= latest

gexp:
	build/env.sh go build -i -v $(shell build/flags.sh) -o $(GOBIN)/gexp ./cmd/gexp
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gexp\" to launch gexp."

gexp-cross: gexp-linux gexp-darwin gexp-windows gexp-android gexp-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gexp-*

gexp-linux: gexp-linux-386 gexp-linux-amd64 gexp-linux-arm
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-*

gexp-linux-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/386 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep 386

gexp-linux-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/amd64 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep amd64

gexp-linux-arm: gexp-linux-arm-5 gexp-linux-arm-6 gexp-linux-arm-7 gexp-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep arm

gexp-linux-arm-5: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/arm-5 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep arm-5

gexp-linux-arm-6: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/arm-6 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep arm-6

gexp-linux-arm-7: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/arm-7 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep arm-7

gexp-linux-arm64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=linux/arm64 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-linux-* | grep arm64

gexp-darwin: gexp-darwin-386 gexp-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gexp-darwin-*

gexp-darwin-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=darwin/386 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-darwin-* | grep 386

gexp-darwin-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=darwin/amd64 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-darwin-* | grep amd64

gexp-windows: gexp-windows-386 gexp-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gexp-windows-*

gexp-windows-386: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=windows/386 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-windows-* | grep 386

gexp-windows-amd64: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=windows/amd64 -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gexp-windows-* | grep amd64

gexp-android: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=android-21/aar -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Android cross compilation done:"
	@ls -ld $(GOBIN)/gexp-android-*

gexp-ios: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=ios-7.0/framework -v $(shell build/flags.sh) ./cmd/gexp
	@echo "iOS framework cross compilation done:"
	@ls -ld $(GOBIN)/gexp-ios-*

evm:
	build/env.sh $(GOROOT)/bin/go install -v $(shell build/flags.sh) ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm to start the evm."

all:
	for cmd in `ls ./cmd/`; do \
		 build/env.sh go build -i -v $(shell build/flags.sh) -o $(GOBIN)/$$cmd ./cmd/$$cmd; \
	done

test: all
	build/env.sh go test ./...

travis-test-with-coverage: all
	build/env.sh go vet ./...
	build/env.sh build/test-global-coverage.sh

xgo:
	build/env.sh go get github.com/karalabe/xgo

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
