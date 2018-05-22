# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: XDC android ios XDC-cross swarm evm all test clean
.PHONY: XDC-linux XDC-linux-386 XDC-linux-amd64 XDC-linux-mips64 XDC-linux-mips64le
.PHONY: XDC-linux-arm XDC-linux-arm-5 XDC-linux-arm-6 XDC-linux-arm-7 XDC-linux-arm64
.PHONY: XDC-darwin XDC-darwin-386 XDC-darwin-amd64
.PHONY: XDC-windows XDC-windows-386 XDC-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

XDC:
	build/env.sh go run build/ci.go install ./cmd/XDC
	@echo "Done building."
	@echo "Run \"$(GOBIN)/XDC\" to launch XDC."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/XDC.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/XDC.framework\" to use the library."

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

XDC-cross: XDC-linux XDC-darwin XDC-windows XDC-android XDC-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/XDC-*

XDC-linux: XDC-linux-386 XDC-linux-amd64 XDC-linux-arm XDC-linux-mips64 XDC-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-*

XDC-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/XDC
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep 386

XDC-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/XDC
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep amd64

XDC-linux-arm: XDC-linux-arm-5 XDC-linux-arm-6 XDC-linux-arm-7 XDC-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep arm

XDC-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/XDC
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep arm-5

XDC-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/XDC
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep arm-6

XDC-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/XDC
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep arm-7

XDC-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/XDC
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep arm64

XDC-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips

XDC-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mipsle

XDC-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips64

XDC-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips64le

XDC-darwin: XDC-darwin-386 XDC-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-*

XDC-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/XDC
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-* | grep 386

XDC-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/XDC
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-* | grep amd64

XDC-windows: XDC-windows-386 XDC-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/XDC-windows-*

XDC-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/XDC
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-windows-* | grep 386

XDC-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/XDC
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-windows-* | grep amd64
