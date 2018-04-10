gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

deps-protocol-muxing: deps
	go get -u github.com/multiformats/go-multicodec
	go get -u github.com/libp2p/go-msgio

deps: gx 
	gx --verbose install --global
	gx-go rewrite

publish:
	gx-go rewrite --undo
