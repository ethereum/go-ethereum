#!/bin/bash

# create random virtual machine test

mkdir --parents ~/testout
cd ~/testout
export EVMJIT="-cache=0"
while [ 1 ]
do	
	TEST="$(docker run --rm --entrypoint=\"/cpp-ethereum/build/test/createRandomStateTest\" ethereum/cppjit-testrunner)"
	# echo "$TEST"
	
	# test pyethereum
	OUTPUT_PYTHON="$(docker run --rm ethereum/python-testrunner --notrace <<< "$TEST")"
	RESULT_PYTHON=$?

	# test go
	OUTPUT_GO="$(docker run --rm ethereum/go-testrunner "$TEST")"
	RESULT_GO=$?
	
	# test cpp-jit
	OUTPUT_CPPJIT="$(docker run --rm ethereum/cppjit-testrunner "$TEST")"
	RESULT_CPPJIT=$?

	# go fails
	if [ "$RESULT_GO" -ne 0 ]; then
		echo Failed:
		echo Output_GO:
		echo $OUTPUT_GO
		echo Test:
		echo "$TEST"
		echo "$TEST" > FailedTest.json
		mv FailedTest.json $(date -d "today" +"%Y%m%d%H%M")GO.json # replace with scp to central server
	fi

	# python fails
	if [ "$RESULT_PYTHON" -ne 0 ]; then
		echo Failed:
		echo Output_PYTHON:
		echo $OUTPUT_PYTHON
		echo Test:
		echo "$TEST"
		echo "$TEST" > FailedTest.json
		mv FailedTest.json $(date -d "today" +"%Y%m%d%H%M")PYTHON.json
	fi

	# cppjit fails
	if [ "$RESULT_CPPJIT" -ne 0 ]; then
		echo Failed:
		echo Output_CPPJIT:
		echo $OUTPUT_CPPJIT
		echo Test:
		echo "$TEST"
		echo "$TEST" > FailedTest.json
		mv FailedTest.json $(date -d "today" +"%Y%m%d%H%M")CPPJIT.json
	fi
done
