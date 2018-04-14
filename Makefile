# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: getf android ios getf-cross swarm evm all test clean
.PHONY: getf-linux getf-linux-386 getf-linux-amd64 getf-linux-mips64 getf-linux-mips64le
.PHONY: getf-linux-arm getf-linux-arm-5 getf-linux-arm-6 getf-linux-arm-7 getf-linux-arm64
.PHONY: getf-darwin getf-darwin-386 getf-darwin-amd64
.PHONY: getf-windows getf-windows-386 getf-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

getf:
	build/env.sh go run build/ci.go install ./cmd/getf
	@echo "Done building."
	@echo "Run \"$(GOBIN)/getf\" to launch getf."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/getf.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

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

getf-cross: getf-linux getf-darwin getf-windows getf-android getf-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/getf-*

getf-linux: getf-linux-386 getf-linux-amd64 getf-linux-arm getf-linux-mips64 getf-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-*

getf-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/getf
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep 386

getf-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/getf
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep amd64

getf-linux-arm: getf-linux-arm-5 getf-linux-arm-6 getf-linux-arm-7 getf-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep arm

getf-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/getf
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep arm-5

getf-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/getf
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep arm-6

getf-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/getf
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep arm-7

getf-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/getf
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep arm64

getf-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/getf
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep mips

getf-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/getf
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep mipsle

getf-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/getf
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep mips64

getf-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/getf
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/getf-linux-* | grep mips64le

getf-darwin: getf-darwin-386 getf-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/getf-darwin-*

getf-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/getf
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/getf-darwin-* | grep 386

getf-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/getf
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/getf-darwin-* | grep amd64

getf-windows: getf-windows-386 getf-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/getf-windows-*

getf-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/getf
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/getf-windows-* | grep 386

getf-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/getf
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/getf-windows-* | grep amd64
