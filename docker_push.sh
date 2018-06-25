#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker push etiennenapoleone/tomochain:latest
docker push etiennenapoleone/tomochain:$(git log --pretty=format:'%h' -n 1 | cat)
