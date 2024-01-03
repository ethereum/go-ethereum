#!/bin/bash

docker build -t go-ethereum-dev -f Dockerfile.dev .

# RUN
# docker run -it go-ethereum-dev sh