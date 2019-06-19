# Release process for Swarm

This document describes the steps required to perform a new release.

## Pre release

1. Make sure that the most recent builds are green and that smoke tests are passing on the cluster.
2. Check if protocols should be bumped. (e.g [PR#1465](https://github.com/ethersphere/swarm/pull/1465))
3. Check if the [CHANGELOG.md](../CHANGELOG.md) is reflecting all the changes.

## Release

1. Create a PR to update `version.go` and `CHANGELOG.md`.  [See example PR](https://github.com/ethersphere/swarm/pull/1469).
2. Merge the PR after all tests passed.
3. Tag the merged commit that went into `master`.
```sh
git checkout master
git pull
git tag v0.4.{x}
git push origin v0.4.{x}
```
4. CI Builds (Travis/Appveyor/DockerHub) will trigger. Wait for them to finish.
5. Verify that the following places have the new release:
     1. [ ] [Website download page](https://ethswarm.org/downloads/)
     2. [ ] [Docker Hub](https://hub.docker.com/r/ethersphere/swarm/tags)
     3. [ ] [Ubuntu PPA](https://launchpad.net/~ethereum/+archive/ubuntu/ethereum/+packages?field.name_filter=ethereum-swarm&field.status_filter=published&field.series_filter=)
6. Create a PR to update `version.go` and `CHANGELOG.md`, this time setting it to `unstable` and increasing the version number:  [See example PR](https://github.com/ethersphere/swarm/pull/1470).
7. Merge the PR after all tests passed.
8. Close the [milestone](https://github.com/ethersphere/swarm/milestones).

## Post release

1. Update bootnodes and the nodes serving swarm-gateways.net
2. Announce the release on social media:
   1. Create post on reddit: https://www.reddit.com/r/ethswarm/
   2. Share post on gitter: [ethersphere/orange-lounge](https://gitter.im/ethersphere/orange-lounge) and [ethereum/swarm](https://gitter.im/ethereum/swarm)
