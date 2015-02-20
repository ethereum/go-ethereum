
gengl: main.go parseqt.go
	go build

%.go: %.rl
	ragel -Z -G2 -o $@ $<

run: gengl
	./gengl
