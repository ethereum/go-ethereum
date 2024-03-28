# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth all test lint clean devtools help

GOBIN = ./build/bin
GO ?= latest
GORUN = go run

#? geth: Build geth
geth:
	$(GORUN) build/ci.go install ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

#? all: Build all packages and executables
all:
	$(GORUN) build/ci.go install

#? test: Run the tests
test: all
	$(GORUN) build/ci.go test

#? lint: Run certain pre-selected linters
lint: ## Run linters.
	$(GORUN) build/ci.go lint

#? clean: Clean go cache, built executables, and the auto generated folder
clean:
	go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

#? devtools: Install recommended developer tools
devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

devnet-up:
	docker-compose -f ./suave/devenv/docker-compose.yml up -d --build

devnet-down:
	docker-compose -f ./suave/devenv/docker-compose.yml down

fmt-contracts:
	cd suave && forge fmt

release:
	docker run \
		--rm \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN="$(GITHUB_TOKEN)" \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-v `pwd`/sysroot:/sysroot \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/goreleaser/goreleaser-cross:v1.21 \
		release --clean --auto-snapshot

#? help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in go-ethereum:"
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'
