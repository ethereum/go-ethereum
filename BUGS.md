# BUGS? THERE ARE NO BUGS!

go test ./tests -count=1 -v -run "^TestBlockchain/ValidBlocks/bcValidBlockTest/SimpleTx3LowS.json$"


run tests under tests

run tests under tests/block_test.go

### Timeout happens automatically when running large block of tests, this is likely why the UI didnt run all tests, it should be configurable in vscode go config as well
go test ./tests -count=1 -timeout 600m -v 2>&1 | Tee-Object run.log