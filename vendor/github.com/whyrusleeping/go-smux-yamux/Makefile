build: deps
	go build ./...

test: deps
	go test ./...

test_race: deps
	go test -race ./...

gx-bins:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

deps: gx-bins
	gx --verbose install --global
	gx-go rewrite

clean: gx-bins
	gx-go rewrite --undo
