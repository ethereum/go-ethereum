VCS_REF=$(git hash-object -t tree /dev/null)
DOCKER_BUILDKIT=1 docker build --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') -t openmev/mev-geth:latest .