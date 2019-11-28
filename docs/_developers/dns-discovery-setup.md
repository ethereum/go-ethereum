---
title: DNS Discovery Setup Guide
sort_key: C
---

This document explains how to set up an [EIP 1459][dns-eip] node list using the devp2p
developer tool. The focus of this guide is creating a public list for the Ethereum mainnet
and public testnets, but you may also find this helpful if you want to set up DNS-based
discovery for a private network.

DNS-based node lists can serve as a fallback option when connectivity to the discovery DHT
is unavailable. In this guide, we'll create node lists by crawling the discovery DHT, then
publishing the resulting node sets under chosen DNS names.

### Installing the devp2p command

cmd/devp2p is a developer utility and is not included in the Geth distribution. You can
install this command using `go get`:

```shell
go get -u github.com/ethereum/go-ethereum/cmd/devp2p
```

To create a signing key, you might also need the `ethkey` utility.

```shell
go get -u github.com/ethereum/go-ethereum/cmd/ethkey
```

### Crawling the v4 DHT

Our first step is to compile a list of all reachable nodes. The DHT crawler in cmd/devp2p
is a batch process which runs for a set amount of time. You should should schedule this command
to run at a regular interval. To create a node list, run

```shell
devp2p discv4 crawl -timeout 30m all-nodes.json
```

This walks the DHT and stores the set of all found nodes in the `all-nodes.json` file.
Subsequent runs of the same command will revalidate previously discovered node records,
add newly-found nodes to the set, and remove nodes which are no longer alive. The quality
of the node set improves with each run because the number of revalidations is tracked
alongside each node in the set.

### Creating sub-lists through filtering

Once `all-nodes.json` has been created and the set contains a sizeable number of nodes,
useful sub-sets of nodes can be extracted using the `devp2p nodeset filter` command. This
command takes a node set file as argument and applies filters given as command-line flags.

To create a filtered node set, first create a new directory to hold the output set. You
can use any directory name, though it's good practice to use the DNS domain name as the
name of this directory.

```shell
mkdir mainnet.nodes.example.org
```

Then, to create the output set containing Ethereum mainnet nodes only, run

```shell
devp2p nodeset filter all-nodes.json -eth-network mainnet > mainnet.nodes.example.org/nodes.json
```

The following filter flags are available:

* `-eth-network ( mainnet | ropsten | rinkeby | goerli )` selects an Ethereum network.
* `-les-server` selects LES server nodes.
* `-ip <mask>` restricts nodes to the given IP range.
* `-min-age <duration>` restricts the result to nodes which have been live for the
  given duration.

### Creating DNS trees

To turn a node list into a DNS node tree, the list needs to be signed. To do this, you
need a key pair. To create the key file in the correct format, you can use the cmd/ethkey
utility. Please choose a good password to encrypt the key on disk.

```shell
ethkey generate dnskey.json
```

Now use `devp2p dns sign` to update the signature of the node list. If your list's
directory name differs from the name you want to publish it at, please specify the DNS
name the using the `-domain` flag. This command will prompt for the key file password and
update the tree signature.

```shell
devp2p dns sign mainnet.nodes.example.org dnskey.json
```

The resulting DNS tree metadata is stored in the
`mainnet.nodes.example.org/enrtree-info.json` file.

### Publishing DNS trees

Now that the tree is signed, it can be published to a DNS provider. cmd/devp2p currently
supports publishing to CloudFlare DNS. You can also export TXT records as a JSON file and
publish them yourself.

To publish to CloudFlare, first create an API token in the management console. cmd/devp2p
expects the API token in the `CLOUDFLARE_API_TOKEN` environment variable. Now use the
following command to upload DNS TXT records via the CloudFlare API:

```shell
devp2p dns to-cloudflare mainnet.nodes.example.org
```

Note that this command uses the domain name specified during signing. Any existing records
below this name will be erased by cmd/devp2p.

[dns-eip]: https://eips.ethereum.org/EIPS/eip-1459
