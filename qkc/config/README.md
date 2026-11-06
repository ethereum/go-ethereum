# QKC deployment configuration

A QKC deployment uses one `cluster.toml` file for the P2P network ID and process
settings: one Master and one Slave per shard. There is no separate `network.toml`
or `--qkc.network` flag.

Each process selects its entry from the deployment. Explicit CLI flags override
process settings. Chain consensus rules belong in Genesis; Genesis loading and
consensus validation are provided separately in [PR #54](https://github.com/QuarkChain/modernized-qkc/pull/54).

## Validate the example deployment

The [example](examples/cluster.toml) defines one Master and two Slaves. From the
repository root:

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

Each command selects a process, applies its overrides, and checks the entire
**deployment** for invalid process settings and conflicting resources. It does
not open Genesis files, check consensus rules or chain membership, initialize
databases, create data directories, or start services. A successful deployment
check does not imply that the chain's Genesis is valid.

Build the standalone CLI with `make qkc`. Replace `go run ./cmd/qkc` with
`build/bin/qkc` in the commands above. Run `checkconfig --help` for supported flags.

## Network ID and chain IDs

| Field | Purpose | Requirement |
| --- | --- | --- |
| `NetworkID` | Identifies the P2P network. | Positive `uint64`, matching geth. |
| `Slaves.ChainID` | Selects a shard process by its EVM chain ID. | Unique positive `uint256`. |

Network IDs and chain IDs are independent. The example uses NetworkID `2` and
chain IDs `100001` and `100002`. Chain IDs range from `1` to `2^256 - 1`; zero is
reserved for root-chain traffic. `Slaves.ChainID` and `--qkc.chain` identify the
same process. Use a quoted decimal string in TOML, such as `ChainID = "100001"`.

Deployment entries determine the process endpoint map. Their order does not
define consensus membership or which chain receives root rewards; those rules
belong in the chains' Genesis configurations.

## Configure processes

Set `NetworkID` at the top level of `cluster.toml`.

- **Master** connects to the external P2P network. Configure process settings
  under `[Master.Node]` and external devp2p under `[Master.Node.P2P]`.
- **Each Slave** runs one shard. Set `ChainID`, and `PrivateHost` / `PrivatePort`
  for internal TCP. Set `PublicHost` / `PublicPort` for HTTP JSON-RPC. Other
  process settings go under `[Slaves.Node]` and execution settings under
  `[Slaves.Eth]`.

These sections reuse [node.Config](../../node/config.go) and
[ethconfig.Config](../../eth/ethconfig/config.go). Omitted settings inherit geth
defaults. The example `DatabaseCache` values illustrate independent per-shard
budgets in MiB.

Use `--qkc.role master` to select Master, or
`--qkc.role slave --qkc.chain <id>` to select a Slave.

### Addresses, storage, and RPC

`PrivateHost` / `PrivatePort` define the internal TCP endpoint used for listening
and dialing by Master and other Slaves. The host must be concrete, and the port
between 1 and 65535. Use an unbracketed IPv6 host, such as `::1`. Master derives
its shard endpoint map from these entries.

`PublicHost` / `PublicPort` map to `Node.HTTPHost` / `Node.HTTPPort`, taking
precedence if both forms appear in TOML. An empty public host disables HTTP;
port zero requests a dynamically assigned port. WebSocket uses `Node.WSHost`
and `Node.WSPort`. Disabling HTTP or WebSocket leaves internal TCP configured.

Give each process its own data directory. Configured directories and IPC
endpoints must be distinct, and enabled TCP listeners must not conflict.

### File references and relative paths

`GenesisFile` is an optional file reference on Master and each Slave. If supplied,
its relative path resolves against `cluster.toml`. This deployment layer records
the path without reading the file or requiring it to exist. The example omits
these references; Genesis JSON examples and their loading follow separately.

Other paths retain geth's handling. Relative `DataDir` resolves from the working
directory. On Unix, a bare IPC filename resolves inside `DataDir`.

## Override process settings

Priority is explicit CLI flags, then the selected deployment entry, then defaults
for that role. Defaults reuse [node.DefaultConfig](../../node/defaults.go) and,
for shards, [ethconfig.Defaults](../../eth/ethconfig/config.go), with these changes:

- Separate data directories under geth's default data directory: `qkc/master`
  and `qkc/shard-<id>`.
- IPC filenames `qkc-master.ipc` and `qkc-shard-<id>.ipc`. On Unix, chain IDs above
  64 bits use a stable short suffix in default directories and IPC filenames.
- Shard RPC module lists include `eth`.
- External P2P is disabled for shards. No Ethereum bootstrap nodes or network
  presets are injected.

HTTP and WebSocket inherit disabled defaults. Use `--http` / `--ws` to enable
an endpoint and `--http=false` / `--ws=false` to disable one. An address flag
only changes an already enabled endpoint; it does not enable one by itself.

Master rejects shard execution flags such as `--syncmode`. Ethereum presets and
unsupported flags are rejected. An explicit `--networkid` or a nonzero local
`Eth.NetworkId` must match the deployment's NetworkID; `--networkid=0` is rejected.
Validation binds each shard's effective `Eth.NetworkId` to that deployment value.

Inline TOML `Eth.Genesis` and local fork overrides are rejected: deployment
settings must not redefine chain consensus rules.

## Use the Go package

1. Call `LoadCluster` to decode deployment settings, apply defaults, and resolve
   any Genesis file references. It does not read Genesis files.
2. Call `SelectNode` with the role and, for a shard, its chain ID.
3. Apply explicit process overrides to the returned `NodeConfig`.
4. Call `Validate(cluster.NetworkID)` on that process, or `cluster.Validate()`
   to check the complete deployment, including resource conflicts.

`SelectNode` returns the chosen entry within the deployment, so its edits are
included in global validation. These checks validate process settings only;
Genesis loading, chain initialization, and service startup are separate steps.

## TOML format

TOML follows geth's case-sensitive Go field names. Unknown fields and duplicate
keys are errors. Omitted settings retain their defaults; explicit zero and false
values remain overrides. There is no shared-fork or network-definition section
in the deployment file.
