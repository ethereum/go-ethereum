# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: tomo android ios tomo-cross swarm evm all test clean
.PHONY: tomo-linux tomo-linux-386 tomo-linux-amd64 tomo-linux-mips64 tomo-linux-mips64le
.PHONY: tomo-linux-arm tomo-linux-arm-5 tomo-linux-arm-6 tomo-linux-arm-7 tomo-linux-arm64
.PHONY: tomo-darwin tomo-darwin-386 tomo-darwin-amd64
.PHONY: tomo-windows tomo-windows-386 tomo-windows-amd64

GOBIN = $(shell pwd)/build/bin
GOFMT = gofmt
GO ?= latest
GO_PACKAGES = .
GO_FILES := $(shell find $(shell go list -f '{{.Dir}}' $(GO_PACKAGES)) -name \*.go)

GIT = git

tomo:
	build/env.sh go run build/ci.go install ./cmd/tomo
	@echo "Done building."
	@echo "Run \"$(GOBIN)/tomo\" to launch tomo."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/tomo.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/tomo.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
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

tomo-cross: tomo-linux tomo-darwin tomo-windows tomo-android tomo-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/tomo-*

tomo-linux: tomo-linux-386 tomo-linux-amd64 tomo-linux-arm tomo-linux-mips64 tomo-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-*

tomo-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/tomo
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep 386

tomo-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/tomo
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep amd64

tomo-linux-arm: tomo-linux-arm-5 tomo-linux-arm-6 tomo-linux-arm-7 tomo-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep arm

tomo-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/tomo
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep arm-5

tomo-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/tomo
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep arm-6

tomo-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/tomo
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep arm-7

tomo-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/tomo
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep arm64

tomo-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/tomo
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep mips

tomo-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/tomo
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep mipsle

tomo-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/tomo
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep mips64

tomo-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/tomo
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/tomo-linux-* | grep mips64le

tomo-darwin: tomo-darwin-386 tomo-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/tomo-darwin-*

tomo-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/tomo
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-darwin-* | grep 386

tomo-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/tomo
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-darwin-* | grep amd64

tomo-windows: tomo-windows-386 tomo-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/tomo-windows-*

tomo-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/tomo
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/tomo-windows-* | grep 386

tomo-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/tomo
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geth-windows-* | grep amd64

gofmt:
	$(GOFMT) -s -w $(GO_FILES)
	$(GIT) checkout vendor
