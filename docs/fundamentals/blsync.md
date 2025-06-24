---
title: Beacon light client
description: Running geth with integrated beacon light client
---

`blsync` is a beacon chain light client. Integrated within Geth, blsync eliminates the necessity of running a separate [consensus client](/docs/getting-started/consensus-clients), making it ideal for use-cases that do not require full validation capabilities. It comes with very low resource requirements and can sync the beacon chain within seconds. `blsync` can be run in two modes: integrated or standalone. In standalone mode it is possible to use it for driving other execution clients.

<note>Important: blsync is not suitable for running a validator. It is also not recommended for nodes handling any amount of money or used in production settings due to its lower security guarantees compared to running a full consensus client.</note>

## Usage

### Integrated mode

To run blsync as part of Geth, you need to specify a public HTTP endpoint and a checkpoint:

- **Choose an Endpoint**: Select a reliable and available endpoint from the [Light Sync Endpoints](https://s1na.github.io/light-sync-endpoints/) list. These nodes are community-maintained.
- **Specify the Checkpoint**: Obtain a weak subjectivity checkpoint from a trusted node operator. The checkpoint should be less than 2 weeks old. Some light sync providers only support checkpoints from the last ~1 hour.

```terminal
geth --beacon.api=<endpoint> --beacon.checkpoint=<checkpoint>
```

#### Checkpoint

A checkpoint is the block root of the first proposed slot of a finalized beacon epoch. It can be found manually or automatically depending on your trust requirements.

##### Retrieve checkpoint automatically

Retrieving the checkpoint and starting Geth at the same time is done using proceedure below.

Replace `<endpoint>` with the chosen *trusted* light sync provider from the list of [endpoints](https://s1na.github.io/light-sync-endpoints/). The following command will run Geth with the integrated blsync. Ensure [`jq`](https://jqlang.org/download/) is installed.

```terminal
BEACON=<endpoint> geth --beacon.api=$BEACON --beacon.checkpoint=$(curl -s $BEACON/eth/v1/beacon/headers/finalized | jq -r ".data.root")
```

##### Manually find a checkpoint

A checkpoint can also be obtained manually. The easiest is to use [beaconcha.in](https://beaconcha.in) and cross check it with another provider like [beaconscan](https://beaconscan.com):

- Visit [beaconcha.in](https://beaconcha.in).
- Navigate to the latest finalized epoch that is ideally 1 hour old.
![Finding a suitable epoch](/images/docs/blsync1.png)
- Open the epoch details and find the first proposed slot at the end of the page.
![Finding the first slot](/images/docs/blsync2.png)
- Compare the block root for the slot against an another source. Verify they are equal.
- Copy the block root field.
![Copy the block root](/images/docs/blsync3.png)
- Fill in the argument for the `--beacon.checkpoint` flag with block root.

###### Example

```terminal
geth --beacon.api=<endpoint> --beacon.checkpoint=<block root>
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

To sync the Sepolia network in this mode, first run Geth:

```terminal
geth --sepolia --datadir light-sepolia-dir
```

The logs will indicate the Engine API path which is by default `http://localhost:8551` and the path to the JWT secret created which is in this instance `./light-sepolia-dir/geth/jwtsecret`. Now blsync can be run:

```terminal
blsync --sepolia --beacon.api https://sepolia.lightclient.xyz --beacon.checkpoint 0x0014732c89a02315d2ada0ed2f63b32ecb8d08751c01bea39011b31ad9ecee36 --blsync.engine.api http://localhost:8551 --blsync.jwtsecret light-sepolia-dir/geth/jwtsecret

INFO [06-23|15:06:33.388] Loaded JWT secret file                   path=light-sepolia-dir/geth/jwtsecret crc32=0x5a92678
INFO [06-23|15:06:34.130] Successful NewPayload                    number=6,169,314 hash=d4204e..772e65 status=SYNCING
INFO [06-23|15:06:34.130] Successful ForkchoiceUpdated             head=d4204e..772e65 status=SYNCING
```
