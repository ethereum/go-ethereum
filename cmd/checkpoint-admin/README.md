## Checkpoint-admin

Checkpoint-admin is a tool for updating checkpoint oracle status. It provides a series of functions including deploying checkpoint oracle contract, signing for new checkpoints, and updating checkpoints in the checkpoint oracle contract.

### Checkpoint

In the LES protocol, there is an important concept called checkpoint. In simple terms, whenever a certain number of blocks are generated on the blockchain, a new checkpoint is generated which contains some important information such as

* Block hash at checkpoint
* Canonical hash trie root at checkpoint
* Bloom trie root at checkpoint

*For a more detailed introduction to checkpoint, please see the LES [spec](https://github.com/ethereum/devp2p/blob/master/caps/les.md).*

Using this information, light clients can skip all historical block headers when synchronizing data and start synchronization from this checkpoint. Therefore, as long as the light client can obtain some latest and correct checkpoints, the amount of data and time for synchronization will be greatly reduced.

However, from a security perspective, the most critical step in a synchronization algorithm based on checkpoints is to determine whether the checkpoint used by the light client is correct. Otherwise, all blockchain data synchronized based on this checkpoint may be wrong. For this we provide two different ways to ensure the correctness of the checkpoint used by the light client.

#### Hardcoded checkpoint

There are several hardcoded checkpoints in the [source code](https://github.com/ethereum/go-ethereum/blob/master/params/config.go#L38) of the go-ethereum project. These checkpoints are updated by go-ethereum developers when new versions of software are released. Because light client users trust Geth developers to some extent, hardcoded checkpoints in the code can also be considered correct.

#### Checkpoint oracle

Hardcoded checkpoints can solve the problem of verifying the correctness of checkpoints (although this is a more centralized solution). But the pain point of this solution is that developers can only update checkpoints when a new version of software is released. In addition, light client users usually do not keep the Geth version they use always up to date. So hardcoded checkpoints used by users are generally stale. Therefore, it still needs to download a large amount of blockchain data during synchronization.

Checkpoint oracle is a more flexible solution. In simple terms, this is a smart contract that is deployed on the blockchain. The smart contract records several designated trusted signers. Whenever enough trusted signers have issued their signatures for the same checkpoint, it can be considered that the checkpoint has been authenticated by the signers. Checkpoints authenticated by trusted signers can be considered correct.

So this way, even without updating the software version, as long as the trusted signers regularly update the checkpoint in oracle on time, the light client can always use the latest and verified checkpoint for data synchronization.

### Usage

Checkpoint-admin is a command line tool designed for checkpoint oracle. Users can easily deploy contracts and update checkpoints through this tool.

#### Install

```shell
go get github.com/ethereum/go-ethereum/cmd/checkpoint-admin
```

#### Deploy

Deploy checkpoint oracle contract. `--signers` indicates the specified trusted signer, and `--threshold` indicates the minimum number of signatures required by trusted signers to update a checkpoint.

```shell
checkpoint-admin deploy --rpc <NODE_RPC_ENDPOINT> --clef <CLEF_ENDPOINT> --signer <SIGNER_TO_SIGN_TX> --signers <TRUSTED_SIGNER_LIST> --threshold 1
```

It is worth noting that checkpoint-admin only supports clef as a signer for transactions and plain text(checkpoint). For more clef usage, please see the clef [tutorial](https://geth.ethereum.org/docs/clef/tutorial) .

#### Sign

Checkpoint-admin provides two different modes of signing. You can automatically obtain the current stable checkpoint and sign it interactively, and you can also use the information provided by the command line flags to sign checkpoint offline.

**Interactive mode**

```shell
checkpoint-admin sign --clef <CLEF_ENDPOINT> --signer <SIGNER_TO_SIGN_CHECKPOINT> --rpc <NODE_RPC_ENDPOINT>
```

*It is worth noting that the connected Geth node can be a fullnode or a light client. If it is fullnode, you must enable the LES protocol. E.G. add `--light.serv 50` to the startup command line flags*.

**Offline mode**

```shell
checkpoint-admin sign --clef <CLEF_ENDPOINT> --signer <SIGNER_TO_SIGN_CHECKPOINT> --index <CHECKPOINT_INDEX> --hash <CHECKPOINT_HASH> --oracle <CHECKPOINT_ORACLE_ADDRESS>
```

*CHECKPOINT_HASH is obtained based on this [calculation method](https://github.com/ethereum/go-ethereum/blob/master/params/config.go#L251).*

#### Publish

Collect enough signatures from different trusted signers for the same checkpoint and submit them to oracle to update the "authenticated" checkpoint in the contract.

```shell
checkpoint-admin publish --clef <CLEF_ENDPOINT> --rpc <NODE_RPC_ENDPOINT> --signer <SIGNER_TO_SIGN_TX> --index <CHECKPOINT_INDEX> --signatures <CHECKPOINT_SIGNATURE_LIST>
```

#### Status query

Check the latest status of checkpoint oracle.

```shell
checkpoint-admin status --rpc <NODE_RPC_ENDPOINT>
```

### Enable checkpoint oracle in your private network

Currently, only the Ethereum mainnet and the default supported test networks (ropsten, rinkeby, goerli) activate this feature. If you want to activate this feature in your private network, you can overwrite the relevant checkpoint oracle settings through the configuration file after deploying the oracle contract.

* Get your node configuration file `geth dumpconfig OTHER_COMMAND_LINE_OPTIONS > config.toml`
* Edit the configuration file and add the following information

```toml
[Eth.CheckpointOracle]
Address = CHECKPOINT_ORACLE_ADDRESS
Signers = [TRUSTED_SIGNER_1, ..., TRUSTED_SIGNER_N]
Threshold = THRESHOLD
```

* Start geth with the modified configuration file

*In the private network, all fullnodes and light clients need to be started using the same checkpoint oracle settings.*