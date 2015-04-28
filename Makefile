# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.
#
# Note that there is no way to run the tests or do anything other than
# building the binaries. This is by design.

.PHONY: geth mist clean
GOBIN = build/bin

geth:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

mist:
	build/env.sh go install -v $(shell build/ldflags.sh) ./cmd/mist
	@echo "Done building."
	@echo "Run \"$(GOBIN)/mist --asset_path=cmd/mist/assets\" to launch mist."

clean:
	rm -fr build/_workspace/pkg/ Godeps/_workspace/pkg $(GOBIN)/*
