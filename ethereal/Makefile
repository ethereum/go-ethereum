UNAME = $(shell uname)

# Default is building
all:
	go install

install:
# Linux build
ifeq ($(UNAME),Linux)
	mkdir -p /usr/local/ethereal
	files=(net.png network.png new.png tx.png)
	for file in "${files[@]}"; do
		cp $file /usr/share/ethereal
	done
	cp -r qml /usr/share/ethereal/qml
	cp $GOPATH/bin/go-ethereum /usr/local/bin/ethereal
endif
# OS X build
ifeq ($(UNAME),Darwin)
	# Execute py script
endif
