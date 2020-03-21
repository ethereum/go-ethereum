# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: bor android ios bor-cross evm all test clean
.PHONY: bor-linux bor-linux-386 bor-linux-amd64 bor-linux-mips64 bor-linux-mips64le
.PHONY: bor-linux-arm bor-linux-arm-5 bor-linux-arm-6 bor-linux-arm-7 bor-linux-arm64
.PHONY: bor-darwin bor-darwin-386 bor-darwin-amd64
.PHONY: bor-windows bor-windows-386 bor-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest
GORUN = go run
GOPATH = $(shell go env GOPATH)

bor:
	$(GORUN) build/ci.go install ./cmd/bor
	mkdir -p $(GOPATH)/bin/
	cp $(GOBIN)/bor $(GOPATH)/bin/
	@echo "Done building."
	@echo "Run \"$(GOBIN)/bor\" to launch bor."

all:
	$(GORUN) build/ci.go install
	mkdir -p $(GOPATH)/bin/
	cp $(GOBIN)/* $(GOPATH)/bin/

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/bor.aar\" to use the library."

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/bor.framework\" to use the library."

test: bor
	go test github.com/maticnetwork/bor/consensus/bor/bor_test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

bor-cross: bor-linux bor-darwin bor-windows bor-android bor-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/bor-*

bor-linux: bor-linux-386 bor-linux-amd64 bor-linux-arm bor-linux-mips64 bor-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-*

bor-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/bor
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep 386

bor-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/bor
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep amd64

bor-linux-arm: bor-linux-arm-5 bor-linux-arm-6 bor-linux-arm-7 bor-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep arm

bor-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/bor
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep arm-5

bor-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/bor
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep arm-6

bor-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/bor
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep arm-7

bor-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/bor
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep arm64

bor-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/bor
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep mips

bor-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/bor
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep mipsle

bor-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/bor
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep mips64

bor-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/bor
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/bor-linux-* | grep mips64le

bor-darwin: bor-darwin-386 bor-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/bor-darwin-*

bor-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/bor
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/bor-darwin-* | grep 386

bor-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/bor
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/bor-darwin-* | grep amd64

bor-windows: bor-windows-386 bor-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/bor-windows-*

bor-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/bor
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/bor-windows-* | grep 386

bor-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/bor
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/bor-windows-* | grep amd64
