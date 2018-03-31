gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

deps: gx
	gx --verbose install --global
	gx-go rewrite

publish:
	gx-go rewrite --undo


SUPPORTED_OS = windows linux darwin freebsd openbsd netbsd
SUPPORTED_ARCH = 386 arm amd64p32 arm64 amd64
XBUILD_TARGETS=$(foreach os,$(SUPPORTED_OS),$(foreach arch,$(SUPPORTED_ARCH),test-xbuild-$(os)/$(arch)))

$(XBUILD_TARGETS): PLATFORM = $(subst /, ,$(patsubst test-xbuild-%,%,$@))
$(XBUILD_TARGETS): GOOS = $(word 1,$(PLATFORM))
$(XBUILD_TARGETS): GOARCH = $(word 2,$(PLATFORM))
$(XBUILD_TARGETS):
	@ if GOOS=$(GOOS) GOARCH=$(GOARCH) go version >/dev/null 2>&1 ; then \
		echo "building $(GOOS)/$(GOARCH)"; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build; \
	fi

test-xbuild: $(XBUILD_TARGETS)
