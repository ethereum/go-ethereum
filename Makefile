UNAME = $(shell uname)
FILES=qml *.png
GOPATH=$(PWD)


# Default is building
all:
	go get -d
	cp *.go $(GOPATH)/src/github.com/ethereum/go-ethereum
	cp -r ui $(GOPATH)/src/github.com/ethereum/go-ethereum
	go build

install:
# Linux build
ifeq ($(UNAME),Linux)
	mkdir -p /usr/share/ethereal
	for file in $(FILES); do \
		cp -r $$file /usr/share/ethereal; \
	done
	cp go-ethereum /usr/local/bin/ethereal
endif
# OS X build
ifeq ($(UNAME),Darwin)
	# Execute py script
endif
