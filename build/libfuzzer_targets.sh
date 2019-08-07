#!/bin/bash
set -x

## go-fuzz doesn't support modules for now, so ensure we do everything
## in the old style GOPATH way
export GO111MODULE="off"

## Install go-fuzz
go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build

## build fuzz targets

TARGETS=("bitutil-fuzzer" "bn256-add-fuzzer" "bn256-mul-fuzzer" "bn256-pair-fuzzer")

go-fuzz-build -libfuzzer -o bitutil-fuzzer.a ./common/bitutil
go-fuzz-build -libfuzzer -func FuzzAdd -o bn256-add-fuzzer.a ./crypto/bn256
go-fuzz-build -libfuzzer -func FuzzMul -o bn256-mul-fuzzer.a ./crypto/bn256
go-fuzz-build -libfuzzer -func FuzzPair -o bn256-pair-fuzzer.a ./crypto/bn256

for TARGET in "${TARGETS[@]}"
do
    clang -fsanitize=fuzzer ${TARGET}.a -o ${TARGET}
done