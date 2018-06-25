#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
docker tag tomochain/tomochain tomochain/tomochain:latest
docker tag tomochain/tomochain tomochain/tomochain:$(git log --pretty=format:'%h' -n 1 | cat)
docker push tomochain/tomochain:latest
docker push tomochain/tomochain:$(git log --pretty=format:'%h' -n 1 | cat)
