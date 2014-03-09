UNAME = $(shell uname)
FILES=qml *.png

# Default is building
all:
	cp *.go $(GOPATH)/src/github.com/ethereum/go-ethereum
	cp -r ui $(GOPATH)/src/github.com/ethereum/go-ethereum
	go install github.com/ethereum/go-ethereum

install:
# Linux build
ifeq ($(UNAME),Linux)
	mkdir /usr/share/ethereal
	for file in $(FILES); do \
		cp -r $$file /usr/share/ethereal; \
	done
	cp $(GOPATH)/bin/go-ethereum /usr/local/bin/ethereal
endif
# OS X build
ifeq ($(UNAME),Darwin)
	# Execute py script
endif
