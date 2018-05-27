# Copyright (C) 2015  Arista Networks, Inc.
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

GO := go
TEST_TIMEOUT := 30s
GOTEST_FLAGS :=

DEFAULT_GOPATH := $${GOPATH%%:*}
GOPATH_BIN := $(DEFAULT_GOPATH)/bin
GOPATH_PKG := $(DEFAULT_GOPATH)/pkg
GOLINT := $(GOPATH_BIN)/golint
GOFOLDERS := find . -type d ! -path "./.git/*"

all: install

install:
	$(GO) install ./...

check: vet test fmtcheck lint

COVER_PKGS := key test
COVER_MODE := count
coverdata:
	echo 'mode: $(COVER_MODE)' >coverage.out
	for dir in $(COVER_PKGS); do \
	  $(GO) test -covermode=$(COVER_MODE) -coverprofile=cov.out-t ./$$dir || exit; \
	  tail -n +2 cov.out-t >> coverage.out && \
	  rm cov.out-t; \
	done;

coverage: coverdata
	$(GO) tool cover -html=coverage.out
	rm -f coverage.out

fmtcheck:
	errors=`gofmt -l .`; if test -n "$$errors"; then echo Check these files for style errors:; echo "$$errors"; exit 1; fi
	find . -name '*.go' ! -name '*.pb.go' -exec ./check_line_len.awk {} +

vet:
	$(GO) vet ./...

lint:
	lint=`$(GOFOLDERS) | xargs -L 1 $(GOLINT) | fgrep -v .pb.go`; if test -n "$$lint"; then echo "$$lint"; exit 1; fi
# The above is ugly, but unfortunately golint doesn't exit 1 when it finds
# lint.  See https://github.com/golang/lint/issues/65

test:
	$(GO) test $(GOTEST_FLAGS) -timeout=$(TEST_TIMEOUT) ./...

docker:
	docker build -f cmd/occlient/Dockerfile .

clean:
	rm -rf $(GOPATH_PKG)/*/github.com/aristanetworks/goarista
	$(GO) clean ./...

.PHONY: all check coverage coverdata docker fmtcheck install lint test vet
