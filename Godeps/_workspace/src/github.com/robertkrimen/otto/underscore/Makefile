.PHONY: source

source: source.go

underscore.js:
	curl -kL http://underscorejs.org/underscore.js > $@

source.go: underscore.js
	go-bindata -f underscore -p underscore -u true < $< 2>/dev/null | grep -v '^//' | gofmt > $@
	head -4 $< >> $@
	mv $< ..
