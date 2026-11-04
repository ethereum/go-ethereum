# QKC configuration

A QKC deployment uses two TOML files:

| File | What it configures | Example |
| --- | --- | --- |
| `network.toml` | Shared network rules: network ID, shards, EVM chain IDs, forks, and consensus parameters. | [network.toml](examples/network.toml) |
| `cluster.toml` | Process settings: one Master and one Slave per shard, with their addresses, data directories, and RPC settings. | [cluster.toml](examples/cluster.toml) |

`cluster.toml` points to the shared network file through `NetworkFile`. Each
process selects its own entry from the deployment. CLI flags can override that
process's settings; network rules remain in the shared file.

## Validate the example deployment

The example defines one Master and two Slaves. From the repository root, check
the configuration for each process:

```sh
# Master
go run ./cmd/qkc checkconfig \
  --config qkc/config/examples/cluster.toml --qkc.role master

# Chain 100001
go run ./cmd/qkc checkconfig \
  --config qkc/config/examples/cluster.toml --qkc.role slave --qkc.chain 100001

# Chain 100002, with its configured HTTP endpoint disabled
go run ./cmd/qkc checkconfig \
  --config qkc/config/examples/cluster.toml --qkc.role slave --qkc.chain 100002 --http=false
```

Each command applies overrides to the selected process, then validates the
**entire deployment**. It does not create data directories, initialize databases,
or start services. Genesis generation and node startup are outside the scope of
`checkconfig`.

To build the standalone CLI, run `make qkc`. You can then replace
`go run ./cmd/qkc` with `build/bin/qkc` in these example commands. Run
`go run ./cmd/qkc checkconfig --help` for the supported flags.

## Configure network rules in `network.toml`

The network file defines the root chain's consensus settings and each shard's
EVM chain ID, forks, and consensus settings. All processes share these rules.

Configure the shared EVM fork heights once under `[ShardForks]`: `HomesteadBlock`,
`EIP150Block`, `EIP155Block`, `EIP158Block`, `ByzantiumBlock`,
`ConstantinopleBlock`, `PetersburgBlock`, `IstanbulBlock`, `BerlinBlock`, and
`LondonBlock`. The loader copies these values into every shard's native
`params.ChainConfig`. Setting any of these fields in `[[Shards]]` is an error,
including when the value matches the shared setting. Nonzero heights are
allowed; omitted fields remain unset, while `"0"` activates a fork at genesis.

### Network ID and chain ID

These IDs serve different purposes:

| Field | Purpose | Requirement |
| --- | --- | --- |
| `NetworkID` | Identifies the P2P network. | Positive `uint32`. |
| `Shards.ChainID` | Identifies a shard for both EVM replay protection and internal routing. | Unique positive `uint256`. |

A network ID may have the same numeric value as a chain ID. The example uses
chain IDs `100001` and `100002`.

Chain IDs range from `1` to `2^256 - 1`; zero is reserved for root-chain
traffic. `Shards.ChainID`, `Slaves.ChainID`, and `--qkc.chain` use the same
value. `NetworkID` is a separate `uint32` value.

The shard count is derived from the `[[Shards]]` entries, and chain IDs may have
gaps. The loader fills each shard's `QuarkChain.ChainIDs` in the same order as
the entries. The first shard receives root rewards and hosts the root staking
contract. Both lists in the example are `[100001, 100002]`. The order is part
of the consensus configuration and must remain unchanged for an initialized
chain. This list cannot be supplied in TOML. There is no separate routing ID.

### PoSW and staking

Configure `Root.PoSW` and every `Shards.QuarkChain.PoSW` independently. Shards do
not inherit the root chain's consensus settings. PoSW controls mining difficulty,
block subsidies, and the optional staking discount. Each section requires:

| Field | Meaning and unit | Valid values |
| --- | --- | --- |
| `AdjustmentCutoff` | Time scale, in seconds, for the PoW difficulty adjustment. | Greater than zero. |
| `AdjustmentFactor` | Divides the parent difficulty to determine the size of one adjustment step. | Greater than zero. |
| `MinimumDifficulty` | Minimum PoW difficulty before staking discounts. | Greater than zero. |
| `CoinbaseAmount` | Initial block subsidy, before decay and reward tax. | Zero or greater; zero disables the subsidy. |
| `RewardTaxRate` | Fraction of the decayed subsidy shared with root; use `"0"` for the root chain. | From `0` to `1`, inclusive. |
| `EpochInterval` | Number of blocks in one reward and staking decay period (epoch). | Greater than zero. |
| `DecayFactor` | Fraction retained each epoch; `"0.88"` retains 88%. | From `0` to `1`, inclusive. |

Amounts (`CoinbaseAmount` and `StakePerBlock`) use the smallest native currency
unit. The example uses zero block subsidies. Both ratios use quoted decimal
strings whose reduced denominator must not exceed 100; for example, `"0.88"`
is the fraction `22/25`.

`StakingEnabled` defaults to `false`, which selects ordinary PoW without a
staking discount. The parameters above are still required. To enable the
discount, set `StakingEnabled = true` and supply positive values for:

| Field | Meaning and unit |
| --- | --- |
| `WindowSize` | Number of blocks in the staking window, including the candidate block. |
| `StakePerBlock` | Initial stake required for one block of staking quota, before decay. |
| `DiffDivider` | Divides the PoW difficulty when a block qualifies for the staking discount. |

For chain migrations, `OriginalBlockNumber` carries over the old chain's height
when calculating reward and staking epochs, even though new block numbers start
at zero. Set `PoSWStakingDecayEnabled = true` to apply the same `EpochInterval`
and `DecayFactor` to `StakePerBlock`; otherwise the stake requirement stays fixed.
These settings do not deploy the staking contract. See
[PoSWConfig](../../params/posw.go) for the full field definitions.

## Configure processes in `cluster.toml`

The deployment must contain each network shard exactly once. Master and Slaves
have different responsibilities:

- **Master** connects to the external P2P network. Configure its process settings
  under `[Master.Node]` and external devp2p under `[Master.Node.P2P]`.
- **Each Slave** runs one shard and has a `[[Slaves]]` entry. Set `ChainID` to its
  shard's `Shards.ChainID` as a quoted decimal string, and `PrivateHost` and
  `PrivatePort` to its internal TCP endpoint. Set `PublicHost`
  and `PublicPort` for client-facing HTTP JSON-RPC. Put other process settings
  under `[Slaves.Node]` and execution settings under `[Slaves.Eth]`.

For field definitions, see [node.Config](../../node/config.go) for the `Node`
sections and [ethconfig.Config](../../eth/ethconfig/config.go) for shard `Eth`
sections.

The example's `DatabaseCache` values illustrate independent per-shard overrides,
in MiB. Omitted execution settings inherit geth defaults.

### Select a process

Use `--qkc.role master` to select Master. Use
`--qkc.role slave --qkc.chain <id>` to select a Slave. The chain selector is
required and uses the same ID as EVM execution and internal routing. For example,
select the chain with `ChainID = "100001"` using `--qkc.chain 100001`.

### Addresses, storage, and RPC

Each Slave's `PrivateHost` and `PrivatePort` define its internal QKC TCP endpoint,
used both for listening and for dialing by Master and other Slaves. For example,
set `PrivateHost = "127.0.0.1"` and `PrivatePort = 38000`. The host must be concrete
and the port must be from 1 to 65535. For IPv6, use an unbracketed host such as
`PrivateHost = "::1"`. Master derives its shard endpoint map from these fields.

`PublicHost` and `PublicPort` configure client-facing HTTP JSON-RPC. Set them
under `[Master]` or in a `[[Slaves]]` entry. They map directly to geth's
`Node.HTTPHost` and `Node.HTTPPort`, taking
precedence if both forms are configured. An empty public host disables HTTP;
port zero requests a dynamically assigned port. Explicit CLI flags apply last.

| Interface | Host | Port | Purpose |
| --- | --- | --- | --- |
| QKC cluster TCP | `PrivateHost` | `PrivatePort` | Master and Slave communication. |
| HTTP JSON-RPC | `PublicHost` | `PublicPort` | Client requests. |
| WebSocket JSON-RPC | `Node.WSHost` | `Node.WSPort` | Client requests and subscriptions. |

Disabling HTTP or WebSocket leaves the internal cluster endpoint configured.

Give each process its own data directory; configured directories must be distinct.
HTTP, WebSocket, and IPC settings are also independent for each process.

### Relative paths

- `NetworkFile = "network.toml"` resolves relative to `cluster.toml`.
- `--qkc.network <file>` replaces `NetworkFile`; a relative override resolves from
  the command's working directory.
- Other paths retain geth's path handling. In particular, a relative `DataDir`
  resolves from the working directory. On Unix, a bare IPC filename such as
  `qkc.ipc` resolves within the process's `DataDir`.

## Override process settings

For process settings, the priority is (highest first):

1. Explicit CLI flags.
2. Values in `cluster.toml` for the selected Master or Shard process.
3. Defaults for the selected role.

### Defaults

Defaults reuse [node.DefaultConfig](../../node/defaults.go) and, for shards,
[ethconfig.Defaults](../../eth/ethconfig/config.go), with these QKC settings:

- Separate data directories under geth's default data directory: `qkc/master`
  and `qkc/shard-<id>`.
- IPC filenames `qkc-master.ipc` and `qkc-shard-<id>.ipc`, so processes keep
  separate endpoints even with empty data directories or Windows named pipes.
- Shard RPC module lists include `eth`.
- External P2P disabled for shards. No Ethereum bootstrap nodes or Ethereum
  network preset are injected.

HTTP and WebSocket inherit geth's disabled defaults. The example explicitly
enables HTTP for both shards; configure each process's RPC endpoints as needed.

### Supported overrides and limits

`checkconfig --help` lists the supported flags for storage, sync mode, P2P,
HTTP/WS/IPC, and RPC limits. Ethereum network presets and unsupported geth flags
are rejected. Master also rejects shard execution flags, such as `--syncmode`.

For HTTP and WebSocket, use `--http` or `--ws` to enable an endpoint and
`--http=false` or `--ws=false` to disable one. An address flag such as
`--http.addr` only changes an already enabled endpoint; by itself it does not
enable a disabled endpoint.

Network rules cannot be overridden per process:

- A shard's effective `Eth.NetworkId` comes from the shared `NetworkID`.
  A nonzero `Eth.NetworkId` in the deployment, or an explicit `--networkid`, must
  match it. `--networkid=0` is rejected.
- Local `Eth.Genesis` and fork overrides are rejected. Define forks in the
  shared network file.

## Use the Go package

To load the same configuration from Go, follow this sequence:

1. Call `LoadCluster` with the deployment file.
2. Optionally replace `ClusterConfig.NetworkFile`.
3. Call `SelectNode` with the role and, for a shard, its routing ID.
4. Apply explicit process overrides to the returned `NodeConfig`.
5. Call `LoadNetwork` with the selected network file.
6. Call `ClusterConfig.Validate` to validate the entire deployment.

`SelectNode` returns the chosen entry within the deployment, so edits to it are
included in validation. Validation also binds each shard's `Eth.NetworkId` to
the shared network ID.

## TOML format

Both files follow geth's conventions:

- Field names are case-sensitive Go field names. Unknown fields and duplicate
  keys are errors.
- `PublicHost` and `PublicPort` are deployment TOML keys for `Node.HTTPHost` and
  `Node.HTTPPort`; the effective configuration keeps geth's field names.
- In both TOML files, shard chain IDs use quoted decimal big-integer values,
  such as `ChainID = "100001"`. Fork block numbers in `network.toml` also use
  quoted big-integer values, such as `LondonBlock = "0"`.
- `MinimumDifficulty`, `CoinbaseAmount`, and `StakePerBlock` use quoted decimal
  or hex integer strings and must fit `uint256`.
- Ratios use quoted decimal strings, such as `RewardTaxRate = "0.5"`.
