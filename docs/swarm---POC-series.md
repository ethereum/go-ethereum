now project features and POC milestones are managed under
https://github.com/ethereum/go-ethereum/projects/6

## POC 0.1: plan bee

* [x] underground dev testnet launched on BZZ day (3.22 is l33t for BZZ)
* [x] create ansible and docker for node deployment
* [x] a few public gateways 

## POC 0.2 sworm 

* [x] rebase on geth 1.5 edge
* [x] small toynet deployed, see [network monitor](http://146.185.130.117/)
* [x] support for separate url schemes for dns enabled, immutable and raw manifest - feature [#2279](https://github.com/ethereum/go-ethereum/issues/2279)
* [x] Ethereum name service integration
* [x] scripted network tests, cluster control framework
* [x] algorithmic improvements on chunker split/join
* [x] algorithmic improvements on upload/download
* [x] algorithmic improvements in kademlia and hive peers manager 
* [x] calibrating kademlia connectivity parameters to toynet scale
* [x] orange paper series released hosted on [swarm toynet](http://swarm-gateways.net/bzz:/swarm#the-thsphr-orange-papers)
* [x] minimalistic [swarm homepage](http://swarm-gateways.net/bzz:/swarm)
* [x] adapt to felix's rpc-client-as-eth-backend scheme to run swarm as a separate daemon 
* [x] merge into main repo develop branch

## POC 0.3 

Priorities are not finalised, this is just a tentative plan.
The following features are prioritised for POC 3 (subject to change)

* [ ] docker on azure: complete test cluster deployment and remote node control framework
* [ ] comprehensive suite of typical network scenarios
* [ ] syncer rewrite - syncdb refactor to storage
* [ ] mist integration
* [ ] storage monitoring and parameter setting API for Mist swarm dashboard
* [ ] bzz protocol should implement info for reporting - feature [#2042](https://github.com/ethereum/go-ethereum/issues/2042)
* [x] improved peer propagation [#2044](https://github.com/ethereum/go-ethereum/issues/2044)
* [x] abstract network simulation framework 
* [x] protocol stack abstraction, pluggable subprotocol components - (swap, hive/protocol, syncer, peers) for pss
* [ ] new p2p API integration - feature [#2060](https://github.com/ethereum/go-ethereum/issues/2060)
* [ ] pss - unicast
* [ ] swap rewrite
* [ ] obfuscation for plausible deniability

## POC 0.4

* [ ] enhanced network monitoring, structured logging and stats aggregation
* [ ] unicast/multicast messaging over swarm - pss 
* [ ] swear and swindle http://swarm-gateways.net/bzz:/swarm/ethersphere/orange-papers/1/sw%5E3.pdf
* [ ] smash/crash proof of custody http://swarm-gateways.net/bzz:/swarm/ethersphere/orange-papers/2/smash.pdf
* [ ] swarm DB support phase 0 - compact manifest trie and proof requests
* [ ] [SWarm On-demand Retrieval Daemon](https://gist.github.com/zelig/aa6eb43615e12d834d9f) - feature [#2049](https://github.com/ethereum/go-ethereum/issues/2049) = sword. ethereum state, contract storage, receipts, blocks on swarm
* [x] ~implement (a reviewed version of) [EIP-26](https://github.com/ethereum/EIPs/issues/26)~ obsoleted by ENS and the vickrey auction
* [ ] swarm namereg/natspec rewrite - enhancement [#2048](https://github.com/ethereum/go-ethereum/issues/2048)
* [x] ~solidity [contractInfo standard](https://github.com/ethereum/solidity/pull/645) and contract source verification support~ https://www.reddit.com/r/ethereum/comments/5d5lyd/first_contract_to_contain_swarm_hash_to_its/

## POC 0.5 