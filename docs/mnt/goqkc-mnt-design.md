# Multi Native Token (MNT) Design in goquarkchain

## Overview

goquarkchain supports multi native token (MNT) — the ability for accounts to hold multiple ERC-20-like tokens at the native protocol level, tracked in the state trie. All token balances (including QKC) are stored in a `TokenBalances` structure within each account.

---

## 1. Account Structure

### goquarkchain Account

**File:** `core/state/state_object.go`

```go
type Account struct {
    Nonce         uint64
    TokenBalances *types.TokenBalances  // optional, replaces single Balance
    Root          common.Hash
    CodeHash      []byte
    FullShardKey  *types.Uint32         // optional, fixed per address
    Optial        []byte                // optional
}
```

**RLP encoding** (6-element list):
```
[nonce, TokenBalances, root(32 bytes), codeHash(32 bytes), FullShardKey(5 bytes), Optional]
```

### Comparison with go-ethereum

| Aspect | go-ethereum | goquarkchain |
|--------|-------------|--------------|
| Elements | 4 | 6 |
| Balance type | `*uint256.Int` (single token) | `*TokenBalances` (multi-token) |
| Balance encoding | 32 bytes, left-aligned big-endian | `0x00` + RLP list, or `0x01` + merkle root |
| Shard key | N/A | 5 bytes (`0x84` + 4 bytes BE uint32) |
| Optional | N/A | N/A |

go-ethereum `StateAccount` for reference:

```go
// core/types/state_account.go
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int   // fixed 32 bytes, left-aligned big-endian
    Root     common.Hash    // 32 bytes
    CodeHash []byte         // 32 bytes (emptyCodeHash if empty)
}
```

---

## 2. Core Types

### 2.1 `core/types/token_balances.go` — TokenBalances

```go
package types

import (
    "bytes"
    "math/big"
    "sort"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/ethereum/go-ethereum/trie"
)

const TokenTrieThreshold = 16

// TokenBalancePair is a single token-balance entry in the RLP list format.
type TokenBalancePair struct {
    TokenID uint64
    Balance *big.Int
}

// TokenBalances holds multiple token balances.
// When the number of non-zero balances <= 16, stores as RLP list.
// When > 16, switches to a SecureTrie for efficient storage.
type TokenBalances struct {
    db      *trie.Database
    trie    *trie.SecureTrie  // nil when using list format
    balances map[uint64]*big.Int
}

// NewEmptyTokenBalances creates an empty TokenBalances with list format.
func NewEmptyTokenBalances() *TokenBalances

// NewTokenBalancesWithMap creates TokenBalances from a map.
func NewTokenBalancesWithMap(data map[uint64]*big.Int) *TokenBalances

// SetValue sets the balance for a specific token ID.
func (t *TokenBalances) SetValue(amount *big.Int, tokenID uint64)

// GetTokenBalance returns the balance for a specific token ID.
func (t *TokenBalances) GetTokenBalance(tokenID uint64) *big.Int

// GetBalanceMap returns a copy of all balances.
func (t *TokenBalances) GetBalanceMap() map[uint64]*big.Int

// Len returns the number of balances in the cache.
func (t *TokenBalances) Len() int

// IsBlank returns true if the balance set is effectively empty.
func (t *TokenBalances) IsBlank() bool

// Commit flushes the in-memory balances to the SecureTrie (if switched).
func (t *TokenBalances) Commit(db *trie.Database)

// SerializeToBytes produces the bytes stored in the account trie.
// Format:
//   - nil: empty balances
//   - 0x00 + RLP list of TokenBalancePair: list format (<=16 tokens)
//   - 0x01 + 32-byte merkle root: trie format (>16 tokens)
func (t *TokenBalances) SerializeToBytes() ([]byte, error)

// EncodeRLP implements rlp.Encoder - wraps SerializeToBytes in RLP string.
func (t *TokenBalances) EncodeRLP(w io.Writer) error

// Copy returns a shallow copy (balances map is shared).
func (t *TokenBalances) Copy() *TokenBalances
```

**Key details**:
- `SerializeToBytes()` for list format: byte `0x00` prefix + RLP-encoded `[]TokenBalancePair` (sorted by TokenID ascending, zero balances excluded)
- `SerializeToBytes()` for trie format: byte `0x01` prefix + 32 bytes of SecureTrie merkle root
- Token IDs in the trie are encoded as 32-byte keys with the uint64 at bytes 24-31 (big-endian)
- Balances in the trie are RLP-encoded `*big.Int` (variable length)

### 2.2 `core/types/uint32_rlp.go` — Uint32 custom RLP type

```go
package types

import (
    "encoding/binary"
    "fmt"
    "io"

    "github.com/ethereum/go-ethereum/rlp"
)

// Uint32 is a fixed 5-byte RLP encoding: 0x84 + 4 bytes big-endian uint32.
// This is QuarkChain's custom encoding for FullShardKey.
type Uint32 uint32

const (
    rlpUint32Prefix = byte(0x84)
    rlpUint32Len    = 5
)

func (u *Uint32) GetValue() uint32
func (u *Uint32) EncodeRLP(w io.Writer) error
func (u *Uint32) DecodeRLP(s *rlp.Stream) error
```

**Encoding format**: `0x84 YY YY YY YY` where YY bytes are the big-endian uint32 value.
- `0x84` means "the following string is 4 bytes" in RLP spec (0x80 + 4); total encoding is 5 bytes (1 prefix + 4 data)

### 2.3 `common/token_codec.go` — Token ID encoding utilities

```go
package common

// 36-based encoding: digits 0-9 (values 0-9), letters A-Z (values 10-35).
// Max token name length: 12 characters (ZZZZZZZZZZZZ).
const (
    TokenBase  = uint64(36)
    TokenIDMax = uint64(4873763662273663091) // ZZZZZZZZZZZZ
)

func TokenIDEncode(str string) uint64
func TokenIDDecode(id uint64) (string, error)
func TokenCharEncode(char byte) uint64
func TokenCharDecode(id uint64) (byte, error)
```

### 2.4 `common/utils.go` — EncodeToByte32

```go
func EncodeToByte32(data uint64) []byte {
    ret := make([]byte, 32)
    binary.BigEndian.PutUint64(ret[24:], data)
    return ret
}
```

This encodes a uint64 token ID into a 32-byte key for the token trie, placing the 8-byte token ID at bytes 24-31.

---

## 3. RLP Encoding Format

### goquarkchain (with MNT)

```
RLP: [nonce_u64, TokenBalances_bytes, root(32 bytes), codehash(32 bytes), shardKey(5 bytes), optional]
     ^ list start
     | nonce: "0a" (for nonce=10)
     | TokenBalances: "c180" (RLP string wrapping nil = empty balances)
     |   or "c1" + 0x00 + RLP list (for ≤16 tokens)
     |   or "c1" + 0x01 + 32 bytes (for >16 tokens, trie format)
     | root: "a0" + 32 bytes
     | codehash: "a0" + 32 bytes
     | shardKey: "84" + 4 bytes big-endian
     └ optional: "80" (nil) or "00" (empty bytes)
```

### TokenBalances serialization formats

| Format | Bytes | Use case |
|--------|-------|----------|
| `nil` | (RLP empty string) | No TokenBalances field set |
| `0x00 + RLP[TokenBalancePair...]` | Variable | ≤16 non-zero token balances |
| `0x01 + 32-byte merkle root` | 33 bytes | >16 non-zero token balances |

### Token ID encoding example

| Name | TokenID (uint64) | 32-byte key (bytes 24-31) |
|------|-------------------|--------------------------|
| "QKC" | 35760 | `00000000000000000000000000008b98` |
| "QKCUP" | 46347397 | `0000000000000000000000000000000000000000000000000000000002c33485` |

---

## 4. MNT Precompile Contracts

### 4.1 Overview

All 5 MNT-specific precompiled contracts are registered in `PrecompiledContractsByzantium` at `core/vm/contracts.go:126-140`. They use the `0x514b43` ("QKC") prefix scheme, consistent with QuarkChain's address naming convention.

| # | Precompile | Address | Gas | Description |
|---|-----------|---------|-----|-------------|
| 1 | `currentMntID` | `0x000000000000000000000000000000514b430001` | 3 | Returns current `transfer_token_id` |
| 2 | `transferMnt` | `0x000000000000000000000000000000514b430002` | dynamic | Transfers token via message call |
| 3 | `deploySystemContract` | `0x000000000000000000000000000000514b430003` | `deployRootChainPoSWStakingContractGas` | Deploys POSW or token manager contracts |
| 4 | `mintMNT` | `0x000000000000000000000000000000514b430004` | 9000 | Mints non-reserved native tokens |
| 5 | `balanceMNT` | `0x000000000000000000000000000000514b430005` | 400 | Queries token balance |

### 4.2 System Contract Addresses (`contracts.go:57-71`)

| Index | Contract | Address | Scope |
|-------|---------|---------|-------|
| 1 | `ROOT_CHAIN_POSW` | `0x514b430000000000000000000000000000000001` | GLOBAL |
| 2 | `NON_RESERVED_NATIVE_TOKEN` | `0x514b430000000000000000000000000000000002` | LOCAL_CHAIN_0 |
| 3 | `GENERAL_NATIVE_TOKEN` | `0x514b430000000000000000000000000000000003` | GLOBAL |

### 4.3 `currentMntID` — Query Current Token ID

**Source:** `contracts.go:515-537`

**Purpose:** Tell the running contract which token it is currently handling. Required by the `TokenIDQueried` enforcement mechanism.

**Input:** None

**Gas:** 3

**Output:** 32 bytes — first 24 bytes zero + last 8 bytes = `transfer_token_id` in big-endian uint64

**Example call:**
```
Target address:  0x000000000000000000000000000000514b430001
Input (calldata): (empty)
Expected output:  0x0000000000000000000000000000000000000000000000000000000002c33485
                                                 (example: QKCUP tokenID = 46347397 = 0x2C33485)
```

**Effect on contract state:** Sets `contract.TokenIDQueried = true`. Without this flag, if the contract receives non-default token value it will revert.

### 4.4 `transferMnt` — Transfer Token via Message Call

**Source:** `contracts.go:539-622`

**Purpose:** Transfer a specific token (not just the default QKC) to another address, optionally with calldata, via a message call.

**Input:** 4 × 32 bytes (96 bytes minimum):

| Offset | Name | Type | Description |
|--------|------|------|-------------|
| 0-31 | `to` | address (padded) | Recipient address |
| 32-63 | `tokenID` | uint256 | Token ID to transfer |
| 64-95 | `value` | uint256 | Amount to transfer |
| 96+ | `data` | bytes | Calldata to pass to recipient |

**Gas:** Computed dynamically: base gas + `CallValueTransferGas` (if value > 0) + `CallNewAccountGas` (if recipient doesn't exist)

**Restrictions:**
- `staticcall` is not allowed
- Cannot call itself (reentrancy guard)
- Token ID must be ≤ `TOKENIDMAX`
- Caller must have sufficient balance

**Example:**
```
Contract calls transferMnt to send 100 tokens of tokenID=5 to 0xAbC...123 with calldata:

Input (4x32 bytes):
  [0x000000000000000000000000abcdef0123456789abcdef0123456789abcdef123]  ← to address
  [0x0000000000000000000000000000000000000000000000000000000000000005]  ← tokenID = 5
  [0x0000000000000000000000000000000000000000000000000000000000000064]  ← value = 100
  [0x00000000000000000000000000000000000000000000000000000000deadbeef...]  ← calldata

Execution flow (contracts.go:555-622):
  1. Parse 4 inputs from calldata
  2. Check staticcall → reject if true
  3. Validate tokenID ≤ TOKENIDMAX
  4. Validate toAddr ≠ transferMnt itself
  5. Compute gasCost = CallValueTransferGas + (if !exist(toAddr) ? CallNewAccountGas : 0)
  6. Check caller has enough balance: GetBalance(caller, tokenID) >= value
  7. Save original evm.TransferTokenID
  8. Set evm.TransferTokenID = tokenID
  9. Call evm.Call() with value + calldata
  10. Apply checkTokenIDQueried on the result
  11. Restore evm.TransferTokenID
  12. Return the inner call's return data

Internally, evm.Call() will:
  - Check CanTransfer(state, caller, value, tokenID=5)
  - CreateAccount(toAddr) if not exists
  - Call Transfer(db, caller, toAddr, value, tokenID=5)
    → db.SubBalance(caller, value, 5)
    → db.AddBalance(toAddr, value, 5)
  - Execute calldata as contract code (if toAddr has code)
```

### 4.5 `mintMNT` — Mint New Native Token

**Source:** `contracts.go:678-734`

**Purpose:** Create (mint) a new non-reserved native token. Only callable by the `NonReservedNativeTokenContract` system contract.

**Input:** 3 × 32 bytes (96 bytes):

| Offset | Name | Type | Description |
|--------|------|------|-------------|
| 0-31 | `minter` | address | Address to receive the minted tokens |
| 32-63 | `tokenID` | uint256 | Token ID (must be > 0 and ≤ TOKENIDMAX, not default) |
| 64-95 | `amount` | uint256 | Amount to mint |

**Gas:** `CallValueTransferGas` (9000)

**Restrictions:**
- Only callable by `NonReservedNativeTokenContract` (`0x514b430002`) — checked via `contract.CallerAddress`
- Only allowed on chain ID 0
- Default token (QKC) cannot be minted
- Amount must be > 0

**Example:**
```
NonReservedNativeTokenContract calls mintMNT:

Input:
  [0x000000000000000000000000abcdef0123456789abcdef0123456789abcdef123]  ← minter address
  [0x0000000000000000000000000000000000000000000000000000000000000138]  ← tokenID = 312 ("MYT" in 36-base)
  [0x0000000000000000000000000000000000000000000000000de0b6b3a7640000]  ← amount = 10^18

Execution flow (contracts.go:695-734):
  1. Parse minter, tokenID, amount
  2. Reject if amount == 0
  3. Validate tokenID ≤ TOKENIDMAX
  4. Check chainID == 0
  5. Check tokenID ≠ default token (QKC)
  6. CreateAccount(minter) if not exists (charges CallNewAccountGas)
  7. Verify caller == NonReservedNativeTokenContract
  8. evm.StateDB.AddBalance(minter, amount, tokenID)
       └─ GetOrNewStateObject(minter)          // 取已有对象（step 6 已建）
       └─ stateObject.AddBalance(amount, tokenID)
            └─ stateObject.SetBalance(previous + amount, tokenID)
                 ├─ journal.append(balanceChange{account: &minter, prev: TokenBalances.GetBalanceMap()})
                 └─ setTokenBalance(newAmount, tokenID)
                      └─ TokenBalances.SetValue(newAmount, tokenID)
  9. Return 0x00...01 (success)
```

### 4.6 `balanceMNT` — Query Token Balance

**Source:** `contracts.go:736-764`

**Input:** 2 × 32 bytes:

| Offset | Name | Type | Description |
|--------|------|------|-------------|
| 0-31 | `addr` | address | Account to query |
| 32-63 | `tokenID` | uint256 | Token ID |

**Gas:** 400

**Output:** 32 bytes containing the balance

**Example:**
```
Query balance of 0xAbC...123 for tokenID=312:

Input:
  [0x000000000000000000000000abcdef0123456789abcdef0123456789abcdef123]  ← address
  [0x0000000000000000000000000000000000000000000000000000000000000138]  ← tokenID = 312

Output: [0x00000000000000000000000000000000000000000000003635c9adc5dea00000]
  (= 10^21 in decimal)

Execution flow (contracts.go:752-764):
  1. Parse addr and tokenID
  2. Validate tokenID ≤ TOKENIDMAX
  3. balance := evm.StateDB.GetBalance(addr, tokenID.Uint64())
  4. Return qCommon.BigToByte32(balance)
```

### 4.7 `deploySystemContract` — Deploy System Contract

**Source:** `contracts.go:624-676`

**Input:** 1 × 32 bytes — contract index

| Index | Contract | Scope | Description |
|-------|---------|-------|-------------|
| 1 | ROOT_CHAIN_POSW | GLOBAL | POSW staking contract |
| 2 | NON_RESERVED_NATIVE_TOKEN | LOCAL_CHAIN_0 | Token manager |
| 3 | GENERAL_NATIVE_TOKEN | GLOBAL | General native token manager |

**Gas:** `deployRootChainPoSWStakingContractGas`

**Restrictions:**
- Contracts with LOCAL_CHAIN_0 scope can only be deployed on chain 0
- Time gate: `evm.Time.Uint64() >= SystemContracts[index].timestamp`
- Uses predetermined address (not nonce-based)

---

## 5. Complete Example: MNT Transaction Flow (Tx → EVM → Balance → Trie)

### Scenario

Account `0xA` (sender) sends 50 QKCUP tokens (tokenID=46347397) to account `0xB` via a standard EVM call. Both `TransferTokenID` and `GasTokenID` are set to QKCUP in the transaction.

### 5.1 Transaction Structure

```
Transaction:
  from:      0xA
  to:        0xB
  value:     50 (in QKCUP, tokenID=46347397)
  gas:       21000
  gas_price: 1000000000
  gas_token_id: 46347397    ← pay gas with QKCUP
  transfer_token_id: 46347397  ← send QKCUP
```

### 5.2 StateTransitionDB Entry Point

**File:** `core/state_transition.go:199-262`

```
st.TransitionDb()
  ├─ 1. st.preCheck()            // verify from has enough balance & nonce
  ├─ 2. IntrinsicGas()           // compute base gas (21000 for simple transfer)
  ├─ 3. st.useGas(gas)           // deduct intrinsic gas
  │
  ├─ contractCreation = (msg.To() == nil) → false
  │
  ├─ 4. st.state.SetNonce(from, getNonce(from) + 1)  // increment nonce
  │
  └─ 5. evm.Call(sender, to(), data, gas, value)
       sender = vm.AccountRef(msg.From())
       to()   = msg.To() = 0xB
       data   = empty
       gas    = remaining gas after intrinsic
       value   = 50
```

### 5.3 EVM.Call — Balance Check and Transfer

**File:** `core/vm/evm.go:198-236`

```
evm.Call(caller=0xA, addr=0xB, input=[], gas=X, value=50)
  │
  ├─ 1. if evm.depth > CallCreateDepth → revert (depth limit)
  │
  ├─ 2. if !evm.Context.CanTransfer(StateDB, 0xA, 50, evm.TransferTokenID=46347397)
  │       → revert
  │
  ├─ 3. if !evm.StateDB.Exist(0xB)
  │       → evm.StateDB.CreateAccount(0xB)    // creates empty account
  │
  ├─ 4. evm.Transfer(StateDB, 0xA, 0xB, value=50, tokenID=46347397)
  │       → core.Transfer(db, sender=0xA, recipient=0xB, amount=50, tokenID=46347397)
  │           → db.SubBalance(0xA, 50, 46347397)      // deduct from sender
  │           → db.AddBalance(0xB, 50, 46347397)      // credit to recipient
  │
  ├─ 5. NewContract(caller, to, value, gas)    // create contract environment
  ├─ 6. SetCallCode(&addr, codeHash, code)
  │
  └─ 7. run(evm, contract, input, readOnly)
       → Since 0xB has no code (just created), interpreter.Run returns STOP
```

### 5.4 SubBalance / AddBalance — Detail

**File:** `core/state/statedb.go:350-364`, `core/state/state_object.go:299-327`

```
Step A: db.SubBalance(0xA, 50, 46347397)
  stateObject := s.GetOrNewStateObject(0xA)   // load from cache or trie
  stateObject.SubBalance(50, 46347397)
    amount.Sign() != 0 → continue
    c.SetBalance(new(big.Int).Sub(c.Balance(46347397), 50), 46347397)
      s.db.journal.append(balanceChange{account: &0xA, prev: TokenBalances.GetBalanceMap()})
      self.setTokenBalance(newBalance, 46347397)
        self.data.TokenBalances.SetValue(newBalance, 46347397)
          // In memory: balances[46347397] = newBalance

Step B: db.AddBalance(0xB, 50, 46347397)
  stateObject := s.GetOrNewStateObject(0xB)   // newly created, empty TokenBalances
  stateObject.AddBalance(50, 46347397)
    c.SetBalance(new(big.Int).Add(c.Balance(46347397), 50), 46347397)
      s.db.journal.append(balanceChange{...})
      self.setTokenBalance(50, 46347397)
        self.data.TokenBalances.SetValue(50, 46347397)
          // In memory: balances[46347397] = 50
```

### 5.5 How Balance Is Stored in TokenBalances

**File:** `core/types/token_balances.go`

```
TokenBalances internal state (in-memory):
  balances = {
    35760:    1000000000  // QKC balance (tokenID=35760)
    46347397: 50          // QKCUP balance (tokenID=46347397)
  }

When serializing to trie (SerializeToBytes):
  Since len(balances) = 2 ≤ 16 → use list format

  serialized = 0x00 || RLP([
    {TokenID: 35760,    Balance: 1000000000},  // sorted by TokenID ascending
    {TokenID: 46347397, Balance: 50},
  ])
```

### 5.6 Trie Update — Commit Phase

**File:** `core/state/statedb.go:692-737`

```
StateDB.Commit(deleteEmptyObjects=true)
  │
  ├─ 1. For each addr in s.journal.dirties → add to s.stateObjectsDirty
  │
  ├─ 2. For each stateObject in s.stateObjects:
  │     │
  │     ├─ if isDirty:
  │     │   │
  │     │   ├─ stateObject.CommitTrie(s.db)     // commit storage trie
  │     │   │   self.data.Root = root            // update storage trie root
  │     │   │
  │     │   └─ s.updateStateObject(stateObject)  // update account in main trie
  │     │       → Account RLP = [
  │     │           nonce,
  │     │           TokenBalances,   // 0x00+list or 0x01+merkle root
  │     │           root,            // 32 bytes (storage trie root)
  │     │           codeHash,        // 32 bytes
  │     │           FullShardKey,    // 5 bytes (0x84 + 4 bytes big-endian)
  │     │           optional,        // nil
  │     │         ]
  │     │       s.trie.TryUpdate(addr[:], data)
  │     │
  │     └─ delete(s.stateObjectsDirty, addr)
  │
  └─ 3. s.trie.Commit() → new state root hash
```

### 5.7 Full Data Flow Summary

```
Transaction (signed, with transfer_token_id=46347397, gas_token_id=46347397)
  │
  ▼
StateTransition.TransitionDb()
  │ preCheck: verify balance(0xA, tokenID=46347397) >= value
  │
  ▼
evm.Call(0xA, 0xB, [], gas, value=50)
  │ CanTransfer(StateDB, 0xA, 50, tokenID=46347397) → true
  │ CreateAccount(0xB)
  │
  ▼
core.Transfer(StateDB, 0xA, 0xB, 50, tokenID=46347397)
  │
  ├─ StateDB.SubBalance(0xA, 50, 46347397)
  │   stateObject(0xA).SubBalance(50, 46347397)
  │   → TokenBalances[46347397] = oldBalance - 50  [in-memory map]
  │   → journal: remember prev balance
  │
  └─ StateDB.AddBalance(0xB, 50, 46347397)
      stateObject(0xB).AddBalance(50, 46347397)
      → TokenBalances[46347397] = 50  [in-memory map, newly created]
      → journal: remember prev balance (empty)
  │
  ▼
StateDB.Commit()
  │
  ├─ updateStateObject(stateObject(0xA))
  │   → rlp.Encode(Account{
  │       Nonce:         oldNonce + 1,
  │       TokenBalances: {balances: {35760: oldQKC, 46347397: oldBalance - 50}},
  │       Root:          storageRoot,
  │       CodeHash:      emptyCodeHash,
  │       FullShardKey:  0,
  │       Optial:        nil,
  │     })
  │   → trie.TryUpdate(keccak(0xA), encodedBytes)
  │
  ├─ updateStateObject(stateObject(0xB))
  │   → rlp.Encode(Account{
  │       Nonce:         0,
  │       TokenBalances: {balances: {46347397: 50}},   // 0xB is a new account with no QKC
  │       Root:          emptyState,
  │       CodeHash:      emptyCodeHash,
  │       FullShardKey:  0,
  │       Optial:        nil,
  │     })
  │   → trie.TryUpdate(keccak(0xB), encodedBytes)
  │
  └─ trie.Commit() → new state root hash
```

---

## 6. Complete Example: Create (Mint) a New MNT Token

### Scenario

The system deploys the `NonReservedNativeTokenContract` and then uses it to mint a new token "MYT" (tokenID=312) with an initial supply of 1,000,000 tokens to account `0xA`.

### 6.1 Step 1: Deploy NonReservedNativeTokenContract

**Trigger:** `deploySystemContract` precompile (index 2)

```
Caller:  coinbase / system account
Target:  0x000000000000000000000000000000514b430003 (deploySystemContract)
Input:   [0x0000000000000000000000000000000000000000000000000000000000000002]  ← index=2
```

**Execution (`contracts.go:639-676`):**
```
deploySystemContract.Run(input, evm, contract):
  1. index = 2
  2. SystemContracts[2] = {
       address:   0x514b430002 (NonReservedNativeTokenContract),
       bytecode:  NonReservedNativeTokenContractBytecode,
       timestamp: math.MaxUint64,
     }
  3. Check SystemContractScopeMap[2] = ContractScopeLocalChain0
  4. Check evm.StateDB.GetChainID() == 0  ← must be chain 0
  5. Check evm.Time.Uint64() >= timestamp (MaxUint64, so effectively always ready)
  6. targetAddr = 0x514b430002 (predetermined)
  7. evm.Create(contract.self, bytecode, contract.Gas, 0, &targetAddr)
     → Sets account at 0x514b430002 with the system contract bytecode
```

### 6.2 Step 2: Mint Tokens via NonReservedNativeTokenContract

**Trigger:** Call to `mintMNT` precompile (index 4) from NonReservedNativeTokenContract

```
Caller:  0x514b430002 (NonReservedNativeTokenContract)
Target:  0x000000000000000000000000000000514b430004 (mintMNT)
Input:   3 × 32 bytes:
         [0x000000000000000000000000abcdef0123456789abcdef0123456789abcdef123]  ← minter = 0xA
         [0x0000000000000000000000000000000000000000000000000000000000000138]  ← tokenID = 312 ("MYT")
         [0x0000000000000000000000000000000000000000000003635c9adc5dea00000]  ← amount = 1,000,000 * 10^12
```

**Execution (`contracts.go:695-734`):**
```
mintMNT.Run(input, evm, contract):
  1-7.  (checks as described in section 4.5)
  8.  evm.StateDB.AddBalance(minter, amount, tokenID)
        ├─ stateObject := s.GetOrNewStateObject(minter)
        ├─ stateObject.AddBalance(amount, tokenID)
        └─ stateObject.SetBalance(10^21, 312)
            ├─ journal.append(balanceChange{account: &minter, prev: {}})
            └─ self.setTokenBalance(10^21, 312)
                → TokenBalances.balances[312] = 10^21
  9.  Return 0x00...01 (success)
```

### 6.3 Step 3: How the Balance Appears in the Trie

After `StateDB.Commit()`:

```
Account at 0xAbC...123 in state trie:

Account RLP = [
  Nonce:         0,
  TokenBalances: (see encoding below),
  Root:          0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421,
  CodeHash:      0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470,
  FullShardKey:  0x8400000000,    // uint32(0)
  Optional:      nil,
]

TokenBalances encoding (1 balance, ≤ 16):
  0x00 || RLP([
    {TokenID: 312, Balance: 10^21},
  ])

  Structure:
    0x00               ← list format prefix
    c? [...]           ← RLP list wrapping 1 TokenBalancePair
      c? [...]         ← RLP list for {TokenID, Balance}
        82 01 38       ← RLP uint: 312 = 0x138 (2-byte integer)
        89 ...         ← RLP uint: 10^21 (9-byte big-endian integer)
```

### 6.4 Step 4: Subsequent Transfer Using the Minted Token

```
Transaction:
  from:      0xAbC...123
  to:        0xDeF...456
  value:     10^18 (1 token with 12 decimals)
  transfer_token_id: 312
  gas_token_id: 312

StateTransition:
  1. preCheck: CanTransfer(StateDB, 0xA, value, 312) → true
  2. evm.Call(0xA, 0xDeF...456, [], gas, value)
     ├─ CanTransfer(StateDB, 0xA, 10^18, 312) → true
     ├─ Transfer(StateDB, 0xA, 0xDeF...456, 10^18, 312)
     │   ├─ SubBalance(0xA, 10^18, 312) → TokenBalances[312] = 10^21 - 10^18
     │   └─ AddBalance(0xDeF...456, 10^18, 312) → TokenBalances[312] = 10^18
     └─ run() → no code at recipient, STOP
```

### 6.5 Calling MNT Precompiles from Solidity

```solidity
// SPDX-License-Identifier: MIT

// Precompile addresses (0x514b43 = "QKC" in the QuarkChain scheme)
address constant CURRENT_MNT_ID = 0x000000000000000000000000000000514b430001;
address constant TRANSFER_MNT   = 0x000000000000000000000000000000514b430002;
address constant MINT_MNT       = 0x000000000000000000000000000000514b430004;
address constant BALANCE_MNT    = 0x000000000000000000000000000000514b430005;

contract MNTUser {
    // Step 1: Query which token this contract is handling
    function getCurrentTokenID() external view returns (uint64) {
        (bool ok, bytes memory result) = CURRENT_MNT_ID.staticcall("");
        require(ok, "currentMntID failed");
        return uint64(bytes8(result[24:]));
    }

    // Step 2: Transfer a specific token to another address
    function transferToken(address to, uint256 tokenID, uint256 value, bytes calldata data) external {
        bytes memory input = abi.encodePacked(
            addressToBytes32(to),
            uint256ToBytes32(tokenID),
            uint256ToBytes32(value),
            data
        );
        (bool ok,) = TRANSFER_MNT.call(input);
        require(ok, "transferMnt failed");
    }

    // Step 3: Check someone's balance for a specific token
    function checkBalance(address addr, uint256 tokenID) external view returns (uint256) {
        bytes memory input = abi.encodePacked(
            addressToBytes32(addr),
            uint256ToBytes32(tokenID)
        );
        (bool ok, bytes memory result) = BALANCE_MNT.staticcall(input);
        require(ok, "balanceMNT failed");
        return uint256(bytes32(result));
    }

    function addressToBytes32(address addr) internal pure returns (bytes32) {
        return bytes32(uint256(uint160(addr)));
    }

    function uint256ToBytes32(uint256 val) internal pure returns (bytes32) {
        return bytes32(val);
    }
}
```

---

## 7. EVM Precompile Dispatch Mechanism

### How the EVM Knows to Call a Precompile

**File:** `core/vm/evm.go:47-70` (the `run()` function)

```go
func run(evm *EVM, contract *Contract, input []byte, readOnly bool) ([]byte, error) {
    // 1. Check if contract.CodeAddr is a precompile address
    if contract.CodeAddr != nil {
        precompiles := PrecompiledContractsByzantium
        if p := precompiles[*contract.CodeAddr]; p != nil &&
           evm.StateDB.GetTimeStamp() > p.GetEnableTime() {
            // → Bypass the bytecode interpreter entirely!
            return RunPrecompiledContract(p, input, contract, evm)
        }
    }
    // 2. If not a precompile, run the bytecode interpreter
    for _, interpreter := range evm.interpreters {
        if interpreter.CanRun(contract.Code) {
            return interpreter.Run(contract, input, readOnly)
        }
    }
    return nil, ErrNoCompatibleInterpreter
}
```

**Key points:**
- `contract.CodeAddr` is set when the EVM resolves the target address of a CALL/CREATE
- The precompile lookup happens **before** the bytecode interpreter
- Each precompile has an `enableTime` — calls before that time fall through to the interpreter
- `RunPrecompiledContract` (contracts.go:142-149) handles gas charging and execution:
  ```go
  func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract, evm *EVM) (ret []byte, err error) {
      gas := p.RequiredGas(input)
      if contract.UseGas(gas) {
          return p.Run(input, evm, contract)
      }
      return nil, ErrOutOfGas
  }
  ```

### Precompile Interface

```go
type PrecompiledContract interface {
    RequiredGas(input []byte) uint64    // gas cost before execution
    Run(input []byte, evm *EVM, contract *Contract) ([]byte, error)  // execution
    GetEnableTime() uint64               // timestamp after which this precompile is active
}
```

### How `ModifyTokenIDQueried` Works

**File:** `core/vm/instructions.go:755-758`

After every CALL/CREATE/DELEGATECALL/STATICCALL, the EVM checks if the target was the `currentMntID` precompile:

```go
func ModifyTokenIDQueried(contract *Contract, toAddr common.Address) {
    if bytes.Equal(toAddr.Bytes(), common.FromHex(currentMntIDAddr)) {
        contract.TokenIDQueried = true
    }
}
```

This is called at the end of `opCall` (line 784), `opCallCode` (line 813), `opDelegateCall` (line 838), and `opStaticCall` (line 864).

### The `TokenIDQueried` Enforcement

**File:** `core/vm/evm.go:186-192`

```go
func checkTokenIDQueried(err error, contract *Contract, txansferTokenID, defaultTokenID uint64) error {
    if err == nil && len(contract.Code) != 0 && !contract.TokenIDQueried &&
        txansferTokenID != defaultTokenID && contract.value.Cmp(new(big.Int)) != 0 {
        err = errExecutionReverted
    }
    return err
}
```

This is called after every:
- `evm.Call()` (line 255 of evm.go)
- `evm.Create()` (line 461 of evm.go)
- `evm.call()` internal (line 286 of evm.go, via callWithGas)

**Meaning:** If a contract receives non-default-token value in a call, it MUST have previously called `currentMntID`. If it hasn't, the call reverts. This prevents accidentally receiving the wrong token type.
