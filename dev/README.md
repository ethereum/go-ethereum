Swarm development environment
=============================

The Swarm development environment is a Linux bash shell which can be run in a
Docker container and provides a predictable build and test environment.

### Start the Docker container

Run the `run.sh` script to build the Docker image and run it, you will then be
at a bash prompt inside the `swarm/dev` directory.

### Build binaries

Run `make` to build the `swarm`, `geth` and `bootnode` binaries into the
`swarm/dev/bin` directory.

### Boot a cluster

Run `make cluster` to start a 3 node Swarm cluster, or run
`scripts/boot-cluster.sh --size N` to boot a cluster of size N.
