#!/bin/bash -x

# creates the necessary docker images to run testrunner.sh locally

docker build --tag="expanse/cppjit-testrunner" docker-cppjit
docker build --tag="expanse/python-testrunner" docker-python
docker build --tag="expanse/go-testrunner" docker-go
