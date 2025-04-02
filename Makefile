# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: XDC XDC-cross evm all test clean
.PHONY: XDC-linux XDC-linux-386 XDC-linux-amd64 XDC-linux-mips64 XDC-linux-mips64le
.PHONY: XDC-darwin XDC-darwin-386 XDC-darwin-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= 1.23.6
GORUN = go run

#? XDC: Build XDC.
XDC:
	go run build/ci.go install ./cmd/XDC
	@echo "Done building."
	@echo "Run \"$(GOBIN)/XDC\" to launch XDC."

XDC-devnet-local:
	@echo "Rebuild the XDC first"
	mv common/constants.go common/constants.go.tmp
	cp common/constants/constants.go.devnet common/constants.go
	make XDC
	rm -rf common/constants.go
	mv common/constants.go.tmp common/constants.go

	@echo "Run the devnet script in local"
	cd cicd/devnet && ./start-local-devnet.sh

bootnode:
	go run build/ci.go install ./cmd/bootnode
	@echo "Done building."
	@echo "Run \"$(GOBIN)/bootnode\" to launch a bootnode."

puppeth:
	go run build/ci.go install ./cmd/puppeth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/puppeth\" to launch puppeth."

#? all: Build all packages and executables.
all:
	go run build/ci.go install

#? test: Run the tests.
test: all
	go run build/ci.go test

#? quick-test: Run the tests except time-consuming tests.
quick-test: all
	go run build/ci.go test --quick

#? lint: Run certain pre-selected linters.
lint: ## Run linters.
	$(GORUN) build/ci.go lint

#? check_tidy: Verify go.mod and go.sum by 'go mod tidy'
check_tidy: ## Run 'go mod tidy'.
	$(GORUN) build/ci.go check_tidy

#? check_generate: Verify everything is 'go generate'-ed
check_generate: ## Run 'go generate ./...'.
	$(GORUN) build/ci.go check_generate

#? fmt: Ensure consistent code formatting.
fmt:
	gofmt -s -w $(shell find . -name "*.go")

#? clean: Clean go cache, built executables, and the auto generated folder.
clean:
	go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

#? devtools: Install recommended developer tools.
devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

#? help: Get more info on make commands.
help: Makefile
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'

# Cross Compilation Targets (xgo)

XDC-cross: XDC-windows-amd64 XDC-darwin-amd64 XDC-linux
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/XDC-*

XDC-linux: XDC-linux-386 XDC-linux-amd64 XDC-linux-mips64 XDC-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-*

XDC-linux-386:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/XDC
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep 386

XDC-linux-amd64:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/XDC
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep amd64

XDC-linux-mips:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips

XDC-linux-mipsle:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mipsle

XDC-linux-mips64:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips64

XDC-linux-mips64le:
	go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/XDC
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/XDC-linux-* | grep mips64le

XDC-darwin: XDC-darwin-386 XDC-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-*

XDC-darwin-386:
	go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/XDC
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-* | grep 386

XDC-darwin-amd64:
	go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/XDC
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-darwin-* | grep amd64

XDC-windows-amd64:
	go run build/ci.go xgo -- --go=$(GO) -buildmode=mode -x --targets=windows/amd64 -v ./cmd/XDC
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/XDC-windows-* | grep amd64
