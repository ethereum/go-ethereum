# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: egem android ios egem-cross swarm evm all test clean
.PHONY: egem-linux egem-linux-386 egem-linux-amd64 egem-linux-mips64 egem-linux-mips64le
.PHONY: egem-linux-arm egem-linux-arm-5 egem-linux-arm-6 egem-linux-arm-7 egem-linux-arm64
.PHONY: egem-darwin egem-darwin-386 egem-darwin-amd64
.PHONY: egem-windows egem-windows-386 egem-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

egem:
	build/env.sh go run build/ci.go install ./cmd/egem
	@echo "Done building."
	@echo "Run \"$(GOBIN)/egem\" to launch egem."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/egem.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/egem.framework\" to use the library."

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

egem-cross: egem-linux egem-darwin egem-windows egem-android egem-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/egem-*

egem-linux: egem-linux-386 egem-linux-amd64 egem-linux-arm egem-linux-mips64 egem-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-*

egem-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/egem
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep 386

egem-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/egem
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep amd64

egem-linux-arm: egem-linux-arm-5 egem-linux-arm-6 egem-linux-arm-7 egem-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep arm

egem-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/egem
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep arm-5

egem-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/egem
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep arm-6

egem-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/egem
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep arm-7

egem-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/egem
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep arm64

egem-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/egem
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep mips

egem-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/egem
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep mipsle

egem-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/egem
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep mips64

egem-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/egem
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/egem-linux-* | grep mips64le

egem-darwin: egem-darwin-386 egem-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/egem-darwin-*

egem-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/egem
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/egem-darwin-* | grep 386

egem-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/egem
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/egem-darwin-* | grep amd64

egem-windows: egem-windows-386 egem-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/egem-windows-*

egem-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/egem
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/egem-windows-* | grep 386

egem-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/egem
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/egem-windows-* | grep amd64
