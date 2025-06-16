---
title: Beacon light client
description: Running geth with integrated beacon light client
---

`blsync` is a beacon chain light client. Integrated within Geth, blsync eliminates the necessity of running a separate [consensus client](/docs/getting-started/consensus-clients), making it ideal for use-cases that do not require full validation capabilities. It comes with very low resource requirements and can sync the beacon chain within seconds. `blsync` can be run in two modes: integrated or standalone. In standalone mode it is possible to use it for driving other execution clients.

<note>Important: blsync is not suitable for running a validator. It is also not recommended for nodes handling any amount of money or used in production settings due to its lower security guarantees compared to running a full consensus client.</note>

## Usage

### Using Dynamic Checkpoint Fetch

This will automatically fetch the latest finalized checkpoint and launch Geth in snap sync mode with light client support (blsync).

Replace `<domain>` with your select beacon api provider from the provided list of [lightsync endpoints](https://s1na.github.io/light-sync-endpoints/). Replace `<checkpoint>` with a url from the list of maintained [checkpoint sync endpoints](https://eth-clients.github.io/checkpoint-sync-endpoints/). The following command can be used to run geth client with blsync directly.

```terminal
export BEACON=<beacon> && \
export CHECKPOINT=<checkpoint> && \
geth --beacon.api=$BEACON --beacon.checkpoint=$(curl -s $CHECKPOINT/checkpointz/v1/status | jq -r '.data.finality.finalized.root')
```

#### Example
```terminal
export BEACON=https://lodestar-sepolia.chainsafe.io && \
export CHECKPOINT=https://sepolia.beaconstate.info && \
geth --sepolia --beacon.api=$BEACON --beacon.checkpoint=$(curl -s $CHECKPOINT/checkpointz/v1/status | jq -r '.data.finality.finalized.root')
```

### Running with Manual Checkpoint Fetch

To run blsync as part of Geth, you need to specify a public HTTP endpoint and a checkpoint:

- **Choose an Endpoint**: Select a reliable and available endpoint from the [Light Sync Endpoints](https://s1na.github.io/light-sync-endpoints/) list. These nodes are community-maintained.

- **Specify the Checkpoint**: Obtain a weak subjectivity checkpoint from a trusted node operator. The checkpoint should be less than 2 weeks old.

#### Checkpoint

A checkpoint is the block root of the first proposed slot of a finalized beacon epoch. In this guide we use [beaconcha.in](https://sepolia.beaconcha.in) to find a checkpoint:

- Visit sepolia.beaconcha.in.
- Navigate to the latest finalized epoch that is ideally 1 hour old.
![Finding a suitable epoch](/images/docs/blsync1.png)
- Open the epoch details and find the first proposed slot at the end of the page.
![Finding the first slot](/images/docs/blsync2.png)
- Copy the block root field.
![Copy the block root](/images/docs/blsync3.png)


## Testing a Beacon API Endpoint

To verify that your Beacon API is reachable and returning valid data, you can use a simple `curl` command to request the light client bootstrap header for a given block root.

Replace `<domain>` with your Beacon API domain, and `<block_hash>` with the hex-encoded block root you want to test.

```terminal
curl -H "Accept: application/json" http://<domain>/eth/v1/beacon/light_client/bootstrap/<block_root>
```

#### Example
```terminal
curl -H "Accept: application/json" http://testing.holesky.beacon-api.nimbus.team/eth/v1/beacon/light_client/bootstrap/0x82e6ba0e288db5eb79c328fc6cb03a6aec921b00af6888bd51d6b000e68e75ac
```

The following command can be used to start Geth with blsync on the Sepolia network. Note that the checkpoint root will be outdated two weeks after the writing of this page and a recent one will have to be picked according to the guide above:

#### Example

```terminal
./build/bin/geth --sepolia --beacon.api https://sepolia.lightclient.xyz --beacon.checkpoint 0x0014732c89a02315d2ada0ed2f63b32ecb8d08751c01bea39011b31ad9ecee36
```

### Running `blsync` as a Standalone Tool

As mentioned before, `blsync` can be run in standalone mode. This will be similar to running a consensus client with low resource requirements and faster sync times. In most cases Geth users can use the integrated mode for convenience. The standalone mode can be used e.g. to drive an execution client other than Geth.

#### Installing

Depending on your [installation method](/docs/getting-started/installing-geth) either you have access to the `blsync` binary or you will have to build it from source by:

```terminal
go build ./cmd/blsync
```

#### Running

Blsync takes the same flags as above to configure the HTTP endpoint as well as checkpoint. It additionally needs flags to connect to the execution client. Specifically `--blsync.engine.api` to configure the Engine API url and `--blsync.jwtsecret` for the JWT authentication token.

Again to sync the Sepolia network in this mode, first run Geth:

```terminal
./build/bin/geth --sepolia --datadir light-sepolia-dir
```

The logs will indicate the Engine API path which is by default `http://localhost:8551` and the path to the JWT secret created which is in this instance `./light-sepolia-dir/geth/jwtsecret`. Now blsync can be run:

```terminal
./blsync --sepolia --beacon.api https://sepolia.lightclient.xyz --beacon.checkpoint 0x0014732c89a02315d2ada0ed2f63b32ecb8d08751c01bea39011b31ad9ecee36 --blsync.engine.api http://localhost:8551 --blsync.jwtsecret light-sepolia-dir/geth/jwtsecret

INFO [06-23|15:06:33.388] Loaded JWT secret file                   path=light-sepolia-dir/geth/jwtsecret crc32=0x5a92678
INFO [06-23|15:06:34.130] Successful NewPayload                    number=6,169,314 hash=d4204e..772e65 status=SYNCING
INFO [06-23|15:06:34.130] Successful ForkchoiceUpdated             head=d4204e..772e65 status=SYNCING
```
