gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

covertools:
	go get github.com/mattn/goveralls
	go get golang.org/x/tools/cmd/cover

deps: gx covertools
	gx --verbose install --global
	gx-go rewrite

publish:
	gx-go rewrite --undo

