.PHONY: tomo tomo-cross evm all test clean
.PHONY: tomo-linux tomo-linux-386 tomo-linux-amd64 tomo-linux-mips64 tomo-linux-mips64le
.PHONY: tomo-darwin tomo-darwin-386 tomo-darwin-amd64

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

tomo-cross: tomo-linux tomo-darwin
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/tomo-*

tomo-linux: tomo-linux-386 tomo-linux-amd64 tomo-linux-mips64 tomo-linux-mips64le
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

gofmt:
	$(GOFMT) -s -w $(GO_FILES)
	$(GIT) checkout vendor
