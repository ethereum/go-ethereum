.PHONY: test fetch clean build err report

TESTER := tester

test: $(TESTER)
	for test in test-*.js; do ./$^ -test=true $$test 1>/dev/null || exit 1; done
	@echo PASS

report: $(TESTER)
	./$^ -report | grep -v "MT READY"

fetch: $(TESTER)
	./$^ fetch

build:
	go build -a -o $(TESTER)

$(TESTER): tester.go
	$(MAKE) build

clean:
	rm -f test-*.js
	rm -f $(TESTER)

err: $(TESTER)
	for test in test-*.js; do ./$^ $$test; done 2>$@
