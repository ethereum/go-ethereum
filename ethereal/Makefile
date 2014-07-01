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
	cp -r assets/* /usr/share/ethereal
	cp go-ethereum /usr/local/bin/ethereal
endif
# OS X build
ifeq ($(UNAME),Darwin)
	# Execute py script
endif
