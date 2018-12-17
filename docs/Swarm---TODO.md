# Sprint plan

# scope
- forwarding only (no recursive lookup and no connecting to new nodes, only working with active peers)

## TODO

- integrate new p2p
- write unit tests for protocol and netstore (without protocol)
- rework protocol errors using errs after PR merged
- integrate new p2p or develop branch after p2p merge
- integrate cademlia into hive / peer pool with new p2p
- work out timeouts and timeout encoding
- cli tools
- url bar and proxy

## CLI 
- hooking into DPA local API
- running as a daemon accepting request via socket?

### - 
## Encryption
- encryption gateway to incentivise encryption of public content
- xor encryption with random chunks
- in-memory encryption keys
- originator encryption for private content 


## APIs
- DAPP API - js integration (Fabian, Alex)
- mist dapp storage scheme, url->hash mapping (Fabian, Alex) https://github.com/ethereum/go-ethereum/wiki/URL-Scheme

# Discuss alternatives 

I suggest we each pick 2/3 and read up on their project status, features, useability, objectives, etc 
- Is it even worth it to reinvent/reimplement the wheel?
- what features do we want now and in future
- roadmap 

# Brainstorming

- storage economy, incentivisation, examples:
-- content owner pays recurring ether fee for storage.
-- scheme to reward content owner each time content is accessed. i.e accessing content would requires fee. this would reward popular content. should be optional though.
- dht  - chain interaction
- proof of custody https://docs.google.com/document/d/1F81ulKEZFPIGNEVRsx0H1gl2YRtf0mUMsX011BzSjnY/edit
- proof of resources http://systemdocs.maidsafe.net/content/system_components/proof_of_resources.html
- nonoutsourceable proofs of storage as mining criteria 
- proof of storage capacity directly rewarded by contract
- streaming, hash chains 
- routing and learning graph traversal
- minimising hops
- forwarding strategies, optimising dispersion of requests 
- lifetime of requests, renewals (repeated retrieval requests), expiry, reposting (repeated storage request)
- redundancy - store same data in multiple nodes (e.g 4x)
- the more accessed a content is, the more available it should be, should increase performance for popular content.

# Simulations

- full table homogeneous nodes network size vs density vs table size expected row-sizes 
- forwarding strategy vs latency vs traffic
- stable table, dropout rate vs routing optimisation by precalculating subtables for all peers. expected distance change (proximity delta) per hop


## Swarm

How far does the analogy go?
    
swarm of bees | a decentralised network of peers
-------|------------
living in a hive | form a distributed preimage archive
where they | where they
gather pollen | gather data chunks which they 
to produce honey | transform into a longer data stream (document)
they consume and store |  they serve and store  
buzzing bzz | using bzz as their communications protocol

