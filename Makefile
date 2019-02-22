# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gubiq android ios gubiq-cross swarm evm all test clean
.PHONY: gubiq-linux gubiq-linux-386 gubiq-linux-amd64 gubiq-linux-mips64 gubiq-linux-mips64le
.PHONY: gubiq-linux-arm gubiq-linux-arm-5 gubiq-linux-arm-6 gubiq-linux-arm-7 gubiq-linux-arm64
.PHONY: gubiq-darwin gubiq-darwin-386 gubiq-darwin-amd64
.PHONY: gubiq-windows gubiq-windows-386 gubiq-windows-amd64

GOBIN = build/bin
GO ?= latest

gubiq:
	build/env.sh go run build/ci.go install ./cmd/gubiq
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gubiq\" to launch gubiq."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

evm:
	build/env.sh go run build/ci.go install ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm\" to start the evm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gubiq.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gubiq.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

# Cross Compilation Targets (xgo)

gubiq-cross: gubiq-linux gubiq-darwin gubiq-windows gubiq-android gubiq-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-*

gubiq-linux: gubiq-linux-386 gubiq-linux-amd64 gubiq-linux-arm gubiq-linux-mips64 gubiq-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-*

gubiq-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gubiq
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep 386

gubiq-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gubiq
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep amd64

gubiq-linux-arm: gubiq-linux-arm-5 gubiq-linux-arm-6 gubiq-linux-arm-7 gubiq-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep arm

gubiq-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gubiq
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep arm-5

gubiq-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gubiq
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep arm-6

gubiq-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gubiq
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep arm-7

gubiq-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gubiq
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep arm64

gubiq-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gubiq
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep mips

gubiq-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gubiq
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep mipsle

gubiq-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gubiq
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep mips64

gubiq-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gubiq
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-linux-* | grep mips64le

gubiq-darwin: gubiq-darwin-386 gubiq-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-darwin-*

gubiq-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gubiq
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-darwin-* | grep 386

gubiq-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gubiq
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-darwin-* | grep amd64

gubiq-windows: gubiq-windows-386 gubiq-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-windows-*

gubiq-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gubiq
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-windows-* | grep 386

gubiq-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gubiq
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubiq-windows-* | grep amd64
