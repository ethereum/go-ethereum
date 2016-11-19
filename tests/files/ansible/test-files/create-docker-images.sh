#!/bin/bash -x

# creates the necessary docker images to run testrunner.sh locally

docker build --tag="ubiq/cppjit-testrunner" docker-cppjit
docker build --tag="ubiq/python-testrunner" docker-python
docker build --tag="ubiq/go-testrunner" docker-go
