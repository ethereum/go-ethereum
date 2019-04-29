.PHONY: XDC XDC-cross evm all test clean
.PHONY: XDC-linux XDC-linux-386 XDC-linux-amd64 XDC-linux-mips64 XDC-linux-mips64le
.PHONY: XDC-darwin XDC-darwin-386 XDC-darwin-amd64

GOBIN = $(shell pwd)/build/bin
GOFMT = gofmt
GO ?= latest
GO_PACKAGES = .
GO_FILES := $(shell find $(shell go list -f '{{.Dir}}' $(GO_PACKAGES)) -name \*.go)

GIT = git

XDC:
	build/env.sh go run build/ci.go install ./cmd/XDC
	@echo "Done building."
	@echo "Run \"$(GOBIN)/XDC\" to launch XDC."

gc:
	build/env.sh go run build/ci.go install ./cmd/gc
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gc\" to launch gc."

bootnode:
	build/env.sh go run build/ci.go install ./cmd/bootnode
	@echo "Done building."
	@echo "Run \"$(GOBIN)/bootnode\" to launch a bootnode."

puppeth:
	build/env.sh go run build/ci.go install ./cmd/puppeth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/puppeth\" to launch puppeth."

all:
	build/env.sh go run build/ci.go install

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# Cross Compilation Targets (xgo)

XDC-cross: XDC-linux XDC-darwin
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/XDC-*

XDC-linux: XDC-linux-386 XDC-linux-amd64 XDC-linux-mips64 XDC-linux-mips64le
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

gofmt:
	$(GOFMT) -s -w $(GO_FILES)
	$(GIT) checkout vendor
