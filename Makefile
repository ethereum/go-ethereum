UNAME = $(shell uname)

# Default is building
all:
	go install

install:
# Linux build
ifeq ($(UNAME),Linux)
	mkdir /usr/local/ethereal
	files=(wallet.qml net.png network.png new.png tx.png)
	for file in "${files[@]}"; do
		cp $file /usr/share/ethereal
	done
	cp $GOPATH/bin/go-ethereum /usr/local/bin/ethereal
endif
# OS X build
ifeq ($(UNAME),Darwin)
	# Execute py script
endif
