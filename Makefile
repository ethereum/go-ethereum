# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth android ios evm all test clean libzkp libzstd

GOBIN = ./build/bin
GO ?= latest
GORUN = go run
LIBSCROLL_ZSTD_VERSION = v0.1.0-rc0-ubuntu20.04
SCROLL_LIB_PATH = /scroll/lib

libzkp:
	cd $(PWD)/rollup/circuitcapacitychecker/libzkp && make libzkp

libzstd:
	@sudo mkdir -p $(SCROLL_LIB_PATH)
	@sudo wget -O $(SCROLL_LIB_PATH)/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$(LIBSCROLL_ZSTD_VERSION)/libscroll_zstd.so

nccc_geth: libzstd ## geth without circuit capacity checker
	@LD_LIBRARY_PATH="$(LD_LIBRARY_PATH):$(SCROLL_LIB_PATH)" CGO_LDFLAGS="-L$(SCROLL_LIB_PATH) -Wl,-rpath,$(SCROLL_LIB_PATH)" $(GORUN) build/ci.go install ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

geth: libzkp libzstd
	@sudo cp $(PWD)/rollup/circuitcapacitychecker/libzkp/libzkp.so $(SCROLL_LIB_PATH)
	@LD_LIBRARY_PATH="$(LD_LIBRARY_PATH):$(SCROLL_LIB_PATH)" CGO_LDFLAGS="-L$(SCROLL_LIB_PATH) -Wl,-rpath,$(SCROLL_LIB_PATH)" $(GORUN) build/ci.go install -buildtags circuit_capacity_checker ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."
	@echo "Import \"$(GOBIN)/geth-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test:
	./run_test.sh

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/kevinburke/go-bindata/go-bindata@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

docker:
	docker build --platform linux/x86_64 -t scrolltech/l2geth:latest ./ -f Dockerfile

mockccc_docker:
	docker build --platform linux/x86_64 -t scrolltech/l2geth:latest ./ -f Dockerfile.mockccc

mockccc_alpine_docker:
	docker build --platform linux/x86_64 -t scrolltech/l2geth:latest ./ -f Dockerfile.mockccc.alpine
