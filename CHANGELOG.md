## v0.4.3 (July 25, 2019)

### Notes

- **Docker users:** The `$PASSWORD` and `$DATADIR` environment variables are not supported anymore since this release. From now on you should mount the password or data dirctories as a volume. For example:
  ```bash
  $ docker run -it -v $PWD/hostdata:/data \
                   -v $PWD/password:/password \
                   ethersphere/swarm:0.4.3 \
                     --datadir /data \
                     --password /password
  ```

### Bug fixes and improvements

* [#1586](https://github.com/ethersphere/swarm/pull/1586): network: structured output for kademlia table
* [#1582](https://github.com/ethersphere/swarm/pull/1582): client: add bzz client, update smoke tests
* [#1578](https://github.com/ethersphere/swarm/pull/1578): swarm-smoke: fix check max prox hosts for pull/push sync modes
* [#1557](https://github.com/ethersphere/swarm/pull/1557): cmd/swarm: allow using a network interface by name for nat purposes
* [#1534](https://github.com/ethersphere/swarm/pull/1534): api, network: count chunk deliveries per peer
* [#1537](https://github.com/ethersphere/swarm/pull/1537): swarm: fix bzz_info.port when using dynamic port allocation
* [#1531](https://github.com/ethersphere/swarm/pull/1531): cmd/swarm: make bzzaccount flag optional and add bzzkeyhex flag
* [#1536](https://github.com/ethersphere/swarm/pull/1536): cmd/swarm: use only one function to parse flags
* [#1530](https://github.com/ethersphere/swarm/pull/1530): network/bitvector: Multibit set/unset + string rep
* [#1555](https://github.com/ethersphere/swarm/pull/1555): PoC: Network simulation framework

## v0.4.2 (June 28, 2019)

### Notes

This release is not backward compatible with the previous versions of Swarm due to changes to the wire protocol of the Retrieve Request messages. Please update your nodes.

### Bug fixes and improvements

* [#1503](https://github.com/ethersphere/swarm/pull/1503): network/simulation: add ExecAdapter capability to swarm simulations
* [#1495](https://github.com/ethersphere/swarm/pull/1495): build: enable ubuntu ppa disco (19.04) builds
* [#1395](https://github.com/ethersphere/swarm/pull/1395): swarm/storage: support for uploading 100gb files
* [#1344](https://github.com/ethersphere/swarm/pull/1344): swarm/network, swarm/storage: simplification of fetchers
* [#1488](https://github.com/ethersphere/swarm/pull/1488): docker: include git commit hash in swarm version

## v0.4.1 (June 13, 2019)

### Improvements

* [#1465](https://github.com/ethersphere/swarm/pull/1465): network: bump proto versions due to change in OfferedHashesMsg
* [#1428](https://github.com/ethersphere/swarm/pull/1428): swarm-smoke: add debug flag
* [#1422](https://github.com/ethersphere/swarm/pull/1422): swarm/network/stream: remove dead code
* [#1463](https://github.com/ethersphere/swarm/pull/1463): docker: create new dockerfiles that are context aware
* [#1466](https://github.com/ethersphere/swarm/pull/1466): changelog for releases

### Bug fixes

* [#1460](https://github.com/ethersphere/swarm/pull/1460): storage: fix alignement panics on 32 bit arch
* [#1422](https://github.com/ethersphere/swarm/pull/1422), [#19650](https://github.com/ethereum/go-ethereum/pull/19650): swarm/network/stream: remove dead code
* [#1420](https://github.com/ethersphere/swarm/pull/1420): swarm, cmd: fix migration link, change loglevel severity
* [#19594](https://github.com/ethereum/go-ethereum/pull/19594): swarm/api/http: fix bzz-hash to return ens resolved hash directly
* [#19599](https://github.com/ethereum/go-ethereum/pull/19599): swarm/storage: fix SubscribePull to not skip chunks

### Notes

* Swarm has split the codebase ([go-ethereum#19661](https://github.com/ethereum/go-ethereum/pull/19661), [#1405](https://github.com/ethersphere/swarm/pull/1405)) from [ethereum/go-ethereum](https://github.com/ethereum/go-ethereum). The code is now under [ethersphere/swarm](https://github.com/ethersphere/swarm)
* New docker images (>=0.4.0) can now be found under https://hub.docker.com/r/ethersphere/swarm

## v0.4.0 (May 17, 2019)

### Changes

* Implemented parallel feed lookups within Swarm Feeds
* Updated syncing protocol subscription algorithm
* Implemented EIP-1577 - Multiaddr support for ENS
* Improved LocalStore implementation
* Added support for syncing tags which provide the ability to measure how long it will take for an uploaded file to sync to the network
* Fixed data race bugs within PSS
* Improved end-to-end integration tests
* Various performance improvements and bug fixes
* Improved instrumentation - metrics and OpenTracing traces

### Notes
This release is not backward compatible with the previous versions of Swarm due to the new LocalStore implementation. If you wish to keep your data, you should run a data migration prior to running this version.

BZZ network ID has been updated to 4.

Swarm v0.4.0 introduces major changes to the existing codebase. Among other things, the storage layer has been rewritten to be more modular and flexible in a manner that will accommodate for our future needs. Since Swarm at this point does not provide any storage guarantees, we have made the decision to not impose any migrations on the nodes that we maintain as part of the public test network, nor on our users. We have provided a [manual](https://github.com/ethersphere/swarm/blob/master/docs/Migration-v0.3-to-v0.4.md) for those of you who are running private deployments and would like to migrate your data to the new local storage schema.
