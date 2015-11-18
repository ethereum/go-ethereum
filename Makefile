# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.


.PHONY: gexp gexp-cross gexp-linux gexp-darwin gexp-windows gexp-android evm all test travis-test-with-coverage xgo clean
GOBIN = build/bin

gexp:
	build/env.sh go install -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gexp\" to launch gexp."

gexp-cross: gexp-linux gexp-darwin gexp-windows gexp-android
	@echo "Full cross compilation done:"
	@ls -l $(GOBIN)/gexp-*

gexp-linux: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=linux/* -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Linux cross compilation done:"
	@ls -l $(GOBIN)/gexp-linux-*

gexp-darwin: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=darwin/* -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Darwin cross compilation done:"
	@ls -l $(GOBIN)/gexp-darwin-*

gexp-windows: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=windows/* -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Windows cross compilation done:"
	@ls -l $(GOBIN)/gexp-windows-*

gexp-android: xgo
	build/env.sh $(GOBIN)/xgo --dest=$(GOBIN) --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 --targets=android-16/*,android-21/* -v $(shell build/flags.sh) ./cmd/gexp
	@echo "Android cross compilation done:"
	@ls -l $(GOBIN)/gexp-android-*

evm:
	build/env.sh $(GOROOT)/bin/go install -v $(shell build/flags.sh) ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm to start the evm."

all:
	build/env.sh go install -v $(shell build/flags.sh) ./...

test: all
	build/env.sh go test ./...

travis-test-with-coverage: all
	build/env.sh build/test-global-coverage.sh

xgo:
	build/env.sh go get github.com/karalabe/xgo

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
