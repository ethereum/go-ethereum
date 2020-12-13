---
title: clique Namespace
sort_key: C
---

The `clique` API provides access to the state of the clique consensus engine. You can use
this API to manage signer votes and to check the health of a private network.

* TOC
{:toc}

### clique_getSnapshot

Retrieves a snapshot of all clique state at a given block.

| Client  | Method invocation                                          |
|:--------|------------------------------------------------------------|
| Console | `clique.getSnapshot(blockNumber)`                          |
| RPC     | `{"method": "clique_getSnapsnot", "params": [blockNumber]}` |

Example:

```javascript
> clique.getSnapshot(5463755)
{
  hash: "0x018194fc50ca62d973e2f85cffef1e6811278ffd2040a4460537f8dbec3d5efc",
  number: 5463755,
  recents: {
    5463752: "0x42eb768f2244c8811c63729a21a3569731535f06",
    5463753: "0x6635f83421bf059cd8111f180f0727128685bae4",
    5463754: "0x7ffc57839b00206d1ad20c69a1981b489f772031",
    5463755: "0xb279182d99e65703f0076e4812653aab85fca0f0"
  },
  signers: {
    0x42eb768f2244c8811c63729a21a3569731535f06: {},
    0x6635f83421bf059cd8111f180f0727128685bae4: {},
    0x7ffc57839b00206d1ad20c69a1981b489f772031: {},
    0xb279182d99e65703f0076e4812653aab85fca0f0: {},
    0xd6ae8250b8348c94847280928c79fb3b63ca453e: {},
    0xda35dee8eddeaa556e4c26268463e26fb91ff74f: {},
    0xfc18cbc391de84dbd87db83b20935d3e89f5dd91: {}
  },
  tally: {},
  votes: []
}
```

### clique_getSnapshotAtHash

Retrieves the state snapshot at a given block.

| Client  | Method invocation                                        |
|:--------|----------------------------------------------------------|
| Console | `clique.getSnapshotAtHash(blockHash)`                    |
| RPC     | `{"method": "clique_getSnapshotAtHash", "params": [blockHash]}` |

### clique_getSigners

Retrieves the list of authorized signers at the specified block.

| Client  | Method invocation                                          |
|:--------|------------------------------------------------------------
| Console | `clique.getSigners(blockNumber)`                           |
| RPC     | `{"method": "clique_getSigners", "params": [blockNumber]}` |

### clique_proposals

Returns the current proposals the node is voting on.

| Client  | Method invocation                              |
|:--------|------------------------------------------------|
| Console | `clique.proposals()`                           |
| RPC     | `{"method": "clique_proposals", "params": []}` |

### clique_propose

Adds a new authorization proposal that the signer will attempt to push through. If the
`auth` parameter is true, the local signer votes for the given address to be included in
the set of authorized signers. With `auth` set to `false`, the vote is against the
address.

| Client  | Method invocation                                         |
|:--------|-----------------------------------------------------------|
| Console | `clique.propose(address, auth)`                           |
| RPC     | `{"method": "clique_propose", "params": [address, auth]}` |

### clique_discard

This method drops a currently running proposal. The signer will not cast
further votes (either for or against) the address.

| Client  | Method invocation                                   |
|:--------|-----------------------------------------------------|
| Console | `clique.discard(address)`                           |
| RPC     | `{"method": "clique_discard", "params": [address]}` |

### clique_status

This is a debugging method which returns statistics about signer activity
for the last 64 blocks. The returned object contains the following fields:

- `inturnPercent`: percentage of blocks signed in-turn
- `sealerActivity`: object containing signer addresses and the number
  of blocks signed by them
- `numBlocks`: number of blocks analyzed

| Client  | Method invocation                                   |
|:--------|-----------------------------------------------------|
| Console | `clique.status()`                                   |
| RPC     | `{"method": "clique_status", "params": [}` |

Example:

```
> clique.status()
{
  inturnPercent: 100,
  numBlocks: 64,
  sealerActivity: {
    0x42eb768f2244c8811c63729a21a3569731535f06: 9,
    0x6635f83421bf059cd8111f180f0727128685bae4: 9,
    0x7ffc57839b00206d1ad20c69a1981b489f772031: 9,
    0xb279182d99e65703f0076e4812653aab85fca0f0: 10,
    0xd6ae8250b8348c94847280928c79fb3b63ca453e: 9,
    0xda35dee8eddeaa556e4c26268463e26fb91ff74f: 9,
    0xfc18cbc391de84dbd87db83b20935d3e89f5dd91: 9
  }
}
```
