#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
docker tag tomochain/tomochain tomochain/tomochain:$1
docker push tomochain/tomochain:$1
