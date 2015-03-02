#!/bin/bash -x

# creates the necessary docker images to run testrunner.sh locally

docker build --tag="ethereum/cppjit-testrunner" docker-cppjit
docker build --tag="ethereum/python-testrunner" docker-python
docker build --tag="ethereum/go-testrunner" docker-go
