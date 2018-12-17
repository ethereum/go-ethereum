# Channels and streams

a *swarm chain* is an ordered list of content that are linked as a forkless chain.
.
This is simply modeled as linked manifests. 

a *channel* is a sequence of manifests (_S_) and a relative path _P_ with a starting manifest _M_ and a streamsize _n_ (can be infinite). A channel is well-formed or regular if in every manifest in the stream _P_ resolves to a consistent mime type _T_ . For instance , if _T_ is `application/bzz-manifest+json`, we say the channel is a _manifest channel_, if the mime-type is `mpeg`, its a video channel. 

A *primary channel* is a channel that actually respect chronological order of creation. 

A *live channel* is a primary channel that keeps updating (adding episodes to the end of the chain)  can have a (semi)-persistent mime-type for  path _P_

A *blockstream channel* is a primary channel provable linked in time via hashes.

A *signed channel* is a primary channel provably linked by signatures (sequence position index signed by the publisher)

*Trackers* are a manifest channel which tracks updates to a primary channel and provides forward linking .

## Example channels:

- name histories, e.g updates of a domain, temporal snapshots of content
- blockchain: blockchain is a special case of blockstream
- git graph, versioning
- modeling a source of information: provable communication with hash chain, not allowed to fork, numbered. 

#### content trackers

reverse index of a stream 
- contains `next` links to following state
- published after next state
- publish provable quality metrics:
- age: starting date of tracker vs date of orig content 
- neg exp forgetting(track date vs primary date of next episode) ~ alertness, puncuality (tracker
- git version control

every named host defines a timeline, 
- create a manifest stream tracking a site

## Ways to link manifests

#### examples

``` json
{ "entries":
  [
    {
      "host": "fefe.eth",
      "number": 9067,
      "previous": "ffca34987",
      "next": "aefbc4569ab",
      "this": "90daefaaabbc",
    }
  ],
  "auth": "3628aeefbc7689523aebc2489",
}
```


# Name resolution 

The host part in a bzz webaddress should be resolved with our version of DNS, ie. using both `NameReg` (name registration contract on ethereum) and a simple mutable storage in swarm. 

## signed version store
The point of channels (https://github.com/ethereum/go-ethereum/wiki/Swarm---Channels
) is to have a total order over a set of manifests.

The typical usecase is that it should be enough to know the name of a site or document to always see the latest version of a software or get the current episode of your favourite series or the consensus state of a blockchain. It should also be possible to deterministically derive the key to future content...

One possibility is to modify the  NameReg entry in the blockchain to point to a swarm hash. Recording each change on the blockchain results in an implicit live channel linking. This scheme is simple inasmuch as it puts authentication completely on the chain. However, it is expensive and not viable given the number of publishers and typical rate of update.

Alternatively, swarm provides the protocol for associating a channel and a position with a swarm hash. The versioning can be authenticated since a message containing host name, sequence position index and swarm hash is signed by the public key registered in ethereum NameReg for the host.

Publishers, fans or paid bookkeepers track updates of a channel and wrap accumulated messages into a tracker manifest. 
Most probably publishers would broadcast updates of a channel manifest in the first place.

This special key-value store can be implemented as a mutable store: the value with a higher index will simply override the previous one.

There can be various standards to derive lookup key deterministically 
the simplest one is `HASH(host:version)` for a specific version and `HASH(host)` for the latest version. 
The content has the following structure:

```
sign[<host>, <version>, <timestamp>, <hash>]
```

Retrieve request for a signed version is the same as a request for a hash. 

    [RetrieveMsg, HASH(host:version), id, timeout, MUTABLE] 
    [RetrieveMsg, HASH(host:0), id, timeout, MUTABLE] 

Store request for a signed version is the same as for a hash:

    [StoreMsg, key, id, MUTABLE, Sign[host, version, time.Unix(), hash]] 

## Format
It is up to debate how we distinguish names to be resolved. 

An early idea was to use a top level domain, such as `.eth` (<source> == `<host>.eth`)
this might limit the possibilities

Another idea was to have it as or part of the protocol: `eth://my-website.home` or `eth+bzz://my-website.home`. This are semantically incorrect, however. 

Third, put an _eth_ inside the host somehow.

Ad-hoc constructs like `bzz://eth:my-website.home` will be rejected by host pattern matchers.

Abusing subdomains `bzz://eth.my-website.home` would cause ambiguity and potential collision.
Abusing auth  `user:pass@my-website.home` would disable basic auth.

A suggestion that most aligns with the *signed versioning* and very simple is that we look up everything that is not a 32 byte hash format for a public key. The version of the site is looked up using the port part of the host. A specific version is given after the `:`. 

The generic pattern then:

```
   (<version_chain>.)<host>(:<number>)(/<path>)
```

### example 0
```
  bzz://breaking.bad.tv/s4/e2/video
```
- _breaking.bad.tv_ is looked up in NameReg to yield public key _P_
- _breaking.bad.tv_ is looked up in the immutable store to yield a message `[_breaking.bad.tv_,0,3,aebf45fbf6ae6aaaafedcbcb467]`
-  `aebf45fbf6ae6aaaafedcbcb467` is looked up in swarm to yield the manifest 
- manifest entry for path `/s4/e2/video` results in the actual document's root key

### example 1
```
  bzz://breaking.bad.tv/video
```
resolves the following way:
- _breaking.bad.tv_ is looked up in NameReg to yield public key _P_
- _breaking.bad.tv_ is looked up in the immutable store to yield - by `H(cookie)` - a message `[_breaking.bad.tv_,s5:e12,3,aebf45fbf6ae6aaaafedcbcb467]` signed by `P`
-  `aebf45fbf6ae6aaaafedcbcb467` is looked up in swarm to yield the manifest 
- manifest entry for path `video` results in the actual document's root key

### example 2
```
  bzz://current.breaking.bad.tv:s4:e10/video
```
- _breaking.bad.tv_ is looked up in NameReg to yield public key _P_
- current.breaking.bad.tv:s4:e10 is looked up in the immutable store to yield a message `[current.breaking.bad.tv,s4:e10,3,45fbf6ae6aaaafedcbcb467ccc]`
-  `45fbf6ae6aaaafedcbcb467ccc` is looked up in swarm to yield the manifest 
- manifest entry for path `video` results in the actual document's root key

### example 3
```
  bzz://breaking.bad.tv/playlist
```
- same as ex 2...
- manifest entry for path `playlist` results in a playlist manifest


### example 4
```
  bzz://stable.ethereum.org:8.1/download/go/mac-os
```
- _stable.ethereum.org_ is looked up in NameReg to yield public key _P_
- stable.ethereum.org:s4:e10 is looked up in the immutable store to yield a message `[stable.ethereum.org,s4:e10,3,45fbf6ae6aaaafedcbcb467ccc]`
-  `45fbf6ae6aaaafedcbcb467ccc` is looked up in swarm to yield the manifest 
- manifest entry for path `video` results in the actual document's root key

Generalised content streaming, subscriptions.
