
# Module XDPoS

## Method XDPoS_getBlockInfoByEpochNum

Parameters:

- epochNumber: integer, required, epoch number

Returns:

result: object EpochNumInfo:

- hash: hash of first block in this epoch
- round: round of epoch
- firstBlock: number of first block in this epoch
- lastBlock: number of last block in this epoch

Example:

```shell
epoch=89300

curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getBlockInfoByEpochNum",
  "params": [
    '"${epoch}"'
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "hash": "0x5a701a8ba642a9b53475bb19cb9a313829f7afb4287caa76bebaea02f0219f89",
    "round": 1,
    "firstBlock": 80370001,
    "lastBlock": 80370838
  }
}
```


## Method XDPoS_getEpochNumbersBetween

Parameters:

- begin: string, required, block number
- end: string, required, block number

Returns:

result: array of uint64

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getEpochNumbersBetween",
  "params": [
    "0x5439860",
    "0x5439c48"
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    88316769
  ]
}
```


## Method XDPoS_getLatestPoolStatus

The `XDPoS_getLatestPoolStatus` method retrieves current vote pool and timeout pool content and missing messages.

Parameters:

None

Returns:

result: object MessageStatus

- vote:    object
- timeout: object

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getLatestPoolStatus"
}' | jq
```

Response:

See [XDPoS_getLatestPoolStatus_response.json](./XDPoS_getLatestPoolStatus_response.json)

## Method XDPoS_getMasternodesByNumber

Parameters:

- number: string, required, BlockNumber

Returns:

result: object MasternodesStatus:

- Number:          uint64
- Round:           uint64
- MasternodesLen:  int
- Masternodes:     array of address
- PenaltyLen:      int
- Penalty:         array of address
- StandbynodesLen: int
- Standbynodes:    array of address
- Error:           string

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getMasternodesByNumber",
  "params": [
    "latest"
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "Epoch": 89300,
    "Number": 0,
    "Round": 0,
    "MasternodesLen": 0,
    "Masternodes": [],
    "PenaltyLen": 0,
    "Penalty": [],
    "StandbynodesLen": 0,
    "Standbynodes": [],
    "Error": null
  }
}
```


## Method XDPoS_getMissedRoundsInEpochByBlockNum

Parameters:

- number: string, required, BlockNumber

Returns:

result: object PublicApiMissedRoundsMetadata:

- EpochRound:       uint64
- EpochBlockNumber: big.Int
- MissedRounds:     array of MissedRoundInfo

MissedRoundInfo:

- Round:            uint64
- Miner:            address
- CurrentBlockHash: hash
- CurrentBlockNum:  big.Int
- ParentBlockHash:  hash
- ParentBlockNum:   big.Int

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getMissedRoundsInEpochByBlockNum",
  "params": [
    "latest"
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "EpochRound": 12134700,
    "EpochBlockNumber": 92336811,
    "MissedRounds": [
      {
        "Round": 12135188,
        "Miner": "0xe230905c99aaa7b68402af8611b89ceda743191e",
        "CurrentBlockHash": "0xbb587da87991d3cb0122e3f79c31202387022343ce4d317545ec4d676f3199fa",
        "CurrentBlockNum": 92337295,
        "ParentBlockHash": "0xd58572701671c51afa1a0f92d8eb3f59e52418f5d3fe1e08765c5b15970c26e3",
        "ParentBlockNum": 92337294
      },
      {
        "Round": 12135108,
        "Miner": "0x5454edee66858dfcc14871cc8b26f57ef528bedc",
        "CurrentBlockHash": "0x94c8ddc157a095f9f726341fdb0d46ccbced72ce352b1d583dc6a32c285c6f9c",
        "CurrentBlockNum": 92337216,
        "ParentBlockHash": "0x4ff8ad273546a982e533f2613e985eb3f5f2606e26d91011d7f555a19e88795b",
        "ParentBlockNum": 92337215
      },
      {
        "Round": 12134762,
        "Miner": "0xbbd2d417a8b6f1b1d7a267cd1d7402b443f35cfe",
        "CurrentBlockHash": "0x9fc78684a5f7fe80d60e3c6f5acf4fd3dde3c96757e1ba18c8f4f13e4e7523d0",
        "CurrentBlockNum": 92336871,
        "ParentBlockHash": "0x0ad9c7848d2464dc3886af083a06dc097f70549a046db8ebd42a233bc02934af",
        "ParentBlockNum": 92336870
      },
      {
        "Round": 12134747,
        "Miner": "0x03d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff",
        "CurrentBlockHash": "0xed4bfc8af63bdd3f9463a03aac3f3f245401a00bf1fdc126aa4f12fbd5fc3b9a",
        "CurrentBlockNum": 92336857,
        "ParentBlockHash": "0xf4ac8db092ee7d4e65d94691af22cf68b024e09c71d71899fec17b266ac16c8e",
        "ParentBlockNum": 92336856
      },
      {
        "Round": 12134704,
        "Miner": "0x2a591f3d64f3ce6b1d2afeead839ad76aab9feb2",
        "CurrentBlockHash": "0x248452a594feb9cd63bfadc5b2d9a4cdaa3c3a5f9e73275171a102693cf9f20c",
        "CurrentBlockNum": 92336815,
        "ParentBlockHash": "0x2e250ff73aab4fc9205c300f544926f7544a6e6193f1a1f5360cc2f8c5f3aaaa",
        "ParentBlockNum": 92336814
      }
    ]
  }
}
```


## Method XDPoS_getSigners

The `getSigners` method retrieves the list of authorized signers at the specified block.

Parameters:

- number: string, required, BlockNumber

Returns:

result: array of address

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getSigners",
  "params": [
    "latest"
  ]
}' | jq
```

Response:

See [XDPoS_getSigners_response.json](./XDPoS_getSigners_response.json)


## Method XDPoS_getSignersAtHash

The `getSignersAtHash` method retrieves the state snapshot at a given block.

Parameters:

- hash: string, required, block hash

Returns:

same as `XDPoS_getSigners`

Example:

```shell
hash=0x5a701a8ba642a9b53475bb19cb9a313829f7afb4287caa76bebaea02f0219f89
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getSignersAtHash",
  "params": [
    "'"${hash}"'"
  ]
}' | jq
```

Response:

See [XDPoS_getSignersAtHash_response.json](./XDPoS_getSignersAtHash_response.json)


## Method XDPoS_getSnapshot

The `getSnapshot` method retrieves the state snapshot at a given block.

Parameters:

- number: string, required, BlockNumber

Returns:

result: object PublicApiSnapshot:

- number:  block number where the snapshot was created
- hash:    block hash where the snapshot was created
- signers: array of authorized signers at this moment
- recents: array of recent signers for spam protections
- votes:   list of votes cast in chronological order
- tally:   current vote tally to avoid recalculating

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getSnapshot",
  "params": [
    "latest"
  ]
}' | jq
```

Response:

See [XDPoS_getSnapshot_response.json](./XDPoS_getSnapshot_response.json)


## Method XDPoS_getSnapshotAtHash

The `getSnapshotAtHash` method retrieves the state snapshot at a given block.

Parameters:

- hash: string, required, block hash

Returns:

same as `XDPoS_getSnapshot`

Example:

```shell
hash=0x5a701a8ba642a9b53475bb19cb9a313829f7afb4287caa76bebaea02f0219f89
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getSnapshotAtHash",
  "params": [
    "'"${hash}"'"
  ]
}' | jq
```

Response:

See [XDPoS_getSnapshotAtHash_response.json](./XDPoS_getSnapshotAtHash_response.json)


## Method XDPoS_getV2BlockByHash

Parameters:

- hash: string, required, block hash

Returns:

same as `XDPoS_getV2BlockByNumber`

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getV2BlockByHash",
  "params": [
    "'"${hash}"'"
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "Hash": "0x5a701a8ba642a9b53475bb19cb9a313829f7afb4287caa76bebaea02f0219f89",
    "Round": 1,
    "Number": 80370001,
    "ParentHash": "0x83b8e385682ca0faa29e0cc4dbf7de08512ec36bc7d4f0cf173ca5a6dbf034dc",
    "Committed": true,
    "Miner": "0x000000000000000000000000047ffe1fc7f6d0b7168c4ccc312221089629f470",
    "Timestamp": 1727714247,
    "EncodedRLP": "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",
    "Error": ""
  }
}
```


## Method XDPoS_getV2BlockByNumber

Parameters:

- number: string, required, BlockNumber

Returns:

result: object V2BlockInfo:

- Hash:       hash
- Round:      uint64
- Number:     big.Int
- ParentHash: hash
- Committed:  bool
- Miner:      common.Hash
- Timestamp:  big.Int
- EncodedRLP: string
- Error:      string

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_getV2BlockByNumber",
  "params": [
    "latest"
  ]
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "Hash": "0x7f809fffd6b2dfb72374be8a58d66a9215e64561fdf907136196108ae8af4113",
    "Round": 11796106,
    "Number": 92002679,
    "ParentHash": "0xb9f8065697575a03dde30fae631b957b4f81cdd0f919a6be558417b57c6461f6",
    "Committed": false,
    "Miner": "0x000000000000000000000000d22fdac1459760f698618d927bbe22249e2b29b9",
    "Timestamp": 1754450490,
    "EncodedRLP": "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",
    "Error": ""
  }
}
```


## Method XDPoS_networkInformation

Parameters:

None

Returns:

result: object NetworkInformation:

- NetworkId:                  big.Int
- XDCValidatorAddress:        address
- RelayerRegistrationAddress: address
- XDCXListingAddress:         address
- XDCZAddress:                address
- LendingAddress:             address
- ConsensusConfigs:           object of XDPoSConfig

Example:

```shell
curl -s -X POST -H "Content-Type: application/json" ${RPC} -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "XDPoS_networkInformation"
}' | jq
```

Response:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "NetworkId": 50,
    "XDCValidatorAddress": "0x0000000000000000000000000000000000000088",
    "RelayerRegistrationAddress": "0x16c63b79f9c8784168103c0b74e6a59ec2de4a02",
    "XDCXListingAddress": "0xde34dd0f536170993e8cff639ddffcf1a85d3e53",
    "XDCZAddress": "0x8c0faeb5c6bed2129b8674f262fd45c4e9468bee",
    "LendingAddress": "0x7d761afd7ff65a79e4173897594a194e3c506e57",
    "ConsensusConfigs": {
      "period": 2,
      "epoch": 900,
      "reward": 5000,
      "rewardCheckpoint": 900,
      "gap": 450,
      "foundationWalletAddr": "0x92a289fe95a85c53b8d0d113cbaef0c1ec98ac65",
      "SkipV1Validation": false,
      "v2": {
        "switchBlock": 80370000,
        "config": {
          "maxMasternodes": 108,
          "switchRound": 3200000,
          "minePeriod": 2,
          "timeoutSyncThreshold": 3,
          "timeoutPeriod": 10,
          "certificateThreshold": 0.667
        },
        "allConfigs": {
          "0": {
            "maxMasternodes": 108,
            "switchRound": 0,
            "minePeriod": 2,
            "timeoutSyncThreshold": 3,
            "timeoutPeriod": 30,
            "certificateThreshold": 0.667
          },
          "2000": {
            "maxMasternodes": 108,
            "switchRound": 2000,
            "minePeriod": 2,
            "timeoutSyncThreshold": 2,
            "timeoutPeriod": 600,
            "certificateThreshold": 0.667
          },
          "220000": {
            "maxMasternodes": 108,
            "switchRound": 220000,
            "minePeriod": 2,
            "timeoutSyncThreshold": 2,
            "timeoutPeriod": 30,
            "certificateThreshold": 0.667
          },
          "3200000": {
            "maxMasternodes": 108,
            "switchRound": 3200000,
            "minePeriod": 2,
            "timeoutSyncThreshold": 3,
            "timeoutPeriod": 10,
            "certificateThreshold": 0.667
          },
          "460000": {
            "maxMasternodes": 108,
            "switchRound": 460000,
            "minePeriod": 2,
            "timeoutSyncThreshold": 2,
            "timeoutPeriod": 20,
            "certificateThreshold": 0.667
          },
          "8000": {
            "maxMasternodes": 108,
            "switchRound": 8000,
            "minePeriod": 2,
            "timeoutSyncThreshold": 2,
            "timeoutPeriod": 60,
            "certificateThreshold": 0.667
          }
        },
        "SkipV2Validation": false
      }
    }
  }
}
```

