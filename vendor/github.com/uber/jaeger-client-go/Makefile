PROJECT_ROOT=github.com/uber/jaeger-client-go
PACKAGES := $(shell glide novendor | grep -v -e ./thrift-gen/... -e ./thrift/...)
# all .go files that don't exist in hidden directories
ALL_SRC := $(shell find . -name "*.go" | grep -v -e vendor -e thrift-gen -e ./thrift/ \
        -e ".*/\..*" \
        -e ".*/_.*" \
        -e ".*/mocks.*")

-include crossdock/rules.mk

export GO15VENDOREXPERIMENT=1

RACE=-race
GOTEST=go test -v $(RACE)
GOLINT=golint
GOVET=go vet
GOFMT=gofmt
FMT_LOG=fmt.log
LINT_LOG=lint.log

THRIFT_VER=0.9.3
THRIFT_IMG=thrift:$(THRIFT_VER)
THRIFT=docker run -v "${PWD}:/data" $(THRIFT_IMG) thrift
THRIFT_GO_ARGS=thrift_import="github.com/apache/thrift/lib/go/thrift"
THRIFT_GEN_DIR=thrift-gen

PASS=$(shell printf "\033[32mPASS\033[0m")
FAIL=$(shell printf "\033[31mFAIL\033[0m")
COLORIZE=sed ''/PASS/s//$(PASS)/'' | sed ''/FAIL/s//$(FAIL)/''

.DEFAULT_GOAL := test-and-lint

.PHONY: test-and-lint
test-and-lint: test fmt lint

.PHONY: test
test:
	bash -c "set -e; set -o pipefail; $(GOTEST) $(PACKAGES) | $(COLORIZE)"

.PHONY: fmt
fmt:
	$(GOFMT) -e -s -l -w $(ALL_SRC)
	./scripts/updateLicenses.sh

.PHONY: lint
lint:
	$(GOVET) $(PACKAGES)
	@cat /dev/null > $(LINT_LOG)
	@$(foreach pkg, $(PACKAGES), $(GOLINT) $(pkg) | grep -v crossdock/thrift >> $(LINT_LOG) || true;)
	@[ ! -s "$(LINT_LOG)" ] || (echo "Lint Failures" | cat - $(LINT_LOG) && false)
	@$(GOFMT) -e -s -l $(ALL_SRC) > $(FMT_LOG)
	./scripts/updateLicenses.sh >> $(FMT_LOG)
	@[ ! -s "$(FMT_LOG)" ] || (echo "go fmt or license check failures, run 'make fmt'" | cat - $(FMT_LOG) && false)


.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
ifeq ($(USE_DEP),true)
	dep ensure
else
	glide install
endif


.PHONY: cover
cover:
	./scripts/cover.sh $(shell go list $(PACKAGES))
	go tool cover -html=cover.out -o cover.html


# This is not part of the regular test target because we don't want to slow it
# down.
.PHONY: test-examples
test-examples:
	make -C examples

# TODO at the moment we're not generating tchan_*.go files
thrift: idl-submodule thrift-image
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/$(THRIFT_GEN_DIR) /data/idl/thrift/agent.thrift
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/$(THRIFT_GEN_DIR) /data/idl/thrift/sampling.thrift
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/$(THRIFT_GEN_DIR) /data/idl/thrift/jaeger.thrift
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/$(THRIFT_GEN_DIR) /data/idl/thrift/zipkincore.thrift
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/$(THRIFT_GEN_DIR) /data/idl/thrift/baggage.thrift
	$(THRIFT) -o /data --gen go:$(THRIFT_GO_ARGS) --out /data/crossdock/thrift/ /data/idl/thrift/crossdock/tracetest.thrift
	sed -i '' 's|"zipkincore"|"$(PROJECT_ROOT)/thrift-gen/zipkincore"|g' $(THRIFT_GEN_DIR)/agent/*.go
	sed -i '' 's|"jaeger"|"$(PROJECT_ROOT)/thrift-gen/jaeger"|g' $(THRIFT_GEN_DIR)/agent/*.go
	sed -i '' 's|"github.com/apache/thrift/lib/go/thrift"|"github.com/uber/jaeger-client-go/thrift"|g' \
		$(THRIFT_GEN_DIR)/*/*.go crossdock/thrift/tracetest/*.go
	rm -rf thrift-gen/*/*-remote
	rm -rf crossdock/thrift/*/*-remote
	rm -rf thrift-gen/jaeger/collector.go

idl-submodule:
	git submodule init
	git submodule update

thrift-image:
	$(THRIFT) -version

.PHONY: install-dep-ci
install-dep-ci:
	- curl -L -s https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 -o $$GOPATH/bin/dep
	- chmod +x $$GOPATH/bin/dep

.PHONY: install-ci
install-ci: install-dep-ci install
	go get github.com/wadey/gocovmerge
	go get github.com/mattn/goveralls
	go get golang.org/x/tools/cmd/cover
	go get github.com/golang/lint/golint

.PHONY: test-ci
test-ci:
	@./scripts/cover.sh $(shell go list $(PACKAGES))
	make lint

