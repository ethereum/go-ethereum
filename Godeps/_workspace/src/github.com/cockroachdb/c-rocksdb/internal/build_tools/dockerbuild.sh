#!/bin/bash
docker run -v $PWD:/rocks -w /rocks buildpack-deps make
