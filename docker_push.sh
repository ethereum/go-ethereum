#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
docker tag tomochain/tomochain tomochain/tomochain:latest
docker tag tomochain/tomochain tomochain/tomochain:$TRAVIS_BUILD_ID
docker push tomochain/tomochain:latest
docker push tomochain/tomochain:$TRAVIS_BUILD_ID
