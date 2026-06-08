# MNT (Multi Native Token) Integration Design for goshard

## Goal

Integrate multi native token support into goshard (a go-ethereum fork used by QuarkChain) such that the resulting **state trie root hash matches exactly** with pyquarkchain/goquarkchain's output. This node type connects via P2P only to pyquarkchain/goquarkchain nodes.

---

## 1. Current State Comparison

### 1.1 goshard (go-ethereum) — StateAccount

**File**: `core/types/state_account.go`

```go
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int    // 256-bit, QKC native token only
    Root     common.Hash     // 32 bytes, storage trie root
    CodeHash []byte          // 32 bytes, emptyCodeHash if empty
}
```

**RLP encoding** (4-element list, via `gen_account_rlp.go`):
```
[nonce, balance(32 bytes), root(32 bytes), codeHash(32 bytes)]
```

### 1.2 goquarkchain — Account

**File**: `core/state/state_object.go`

```go
type Account struct {
    Nonce         uint64
    TokenBalances *types.TokenBalances  // optional, multi-token balances
    Root          common.Hash
    CodeHash      []byte
    FullShardKey  *types.Uint32         // optional, 5-byte custom RLP
    Optial        []byte                // optional
}
```

**RLP encoding** (6-element list):
```
[nonce, TokenBalances, root(32 bytes), codeHash(32 bytes), FullShardKey(5 bytes), Optional]
```

### 1.3 Key Difference

| Aspect | goshard | goquarkchain |
|--------|---------|--------------|
| Elements | 4 | 6 |
| Balance type | `*uint256.Int` (QKC only) | `*TokenBalances` (multi-token, `*uint256.Int`) |
| Balance encoding in RLP | 32 bytes, left-aligned big-endian | `0x00` + RLP list, or `0x01` + merkle root |
| Shard key | N/A | 5 bytes (`0x84` + 4 bytes BE uint32) |
| Optional | N/A | N/A |

---

## 2. goshard StateAccount Design

### 2.1 StateAccount Struct

goshard's `StateAccount` extends the original go-ethereum 4-field struct with a new `MntBalances` field:

**File**: `core/types/state_account.go`

```go
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int             // 256-bit, QKC native token only
    Root     common.Hash              // 32 bytes, storage trie root
    CodeHash []byte                   // 32 bytes, emptyCodeHash if empty
    MntBalances *types.TokenBalances  // non-QKC MNT balances; nil = no MNT tokens
}
```

### 2.2 RLP Encoding (QuarkChain Compatible)

goshard's `StateAccount` does **not** use go-ethereum's standard 4-element RLP (`gen_account_rlp.go`) for state trie storage. Instead, `updateStateObject` calls a custom encoder that produces a **6-element RLP** byte-for-byte identical to goquarkchain:

```
[Nonce, MntBalances_encoded, Root(32 bytes), CodeHash(32 bytes), FullShardKey(5 bytes), Optional]
```

Mapping to goquarkchain `Account` RLP:

| Element | goshard source | goquarkchain source |
|---------|---------------|---------------------|
| Nonce | `StateAccount.Nonce` | `Account.Nonce` |
| TokenBalances | `StateAccount.Balance` (QKC, tokenID=35760) + `StateAccount.MntBalances` (non-QKC), merged at encode time | `Account.TokenBalances` |
| Root | `StateAccount.Root` | `Account.Root` |
| CodeHash | `StateAccount.CodeHash` | `Account.CodeHash` |
| FullShardKey | derived from Config at encode time (not stored in struct) | `Account.FullShardKey` |
| Optional | `nil` | `Account.Optial` |

At encode time, `EncodeAccountRLP` merges `Balance` (QKC, tokenID=35760) into `MntBalances` before serializing:

```
mergedTokenBalances = MntBalances ∪ {tokenID=35760: Balance}
```

Serialization format matches goquarkchain `TokenBalances` exactly:
- empty (no tokens) → RLP empty string `0x80`
- ≤ 16 non-zero balances → `0x00` prefix + RLP list of `TokenBalancePair` (sorted by TokenID ascending)
- \> 16 non-zero balances → `0x01` prefix + 32-byte SecureTrie merkle root

---

## 3. Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│  goshard (geth fork) with MNT                        │
│                                                      │
│  Internal:  StateAccount extended (5 fields)         │
│            Balance = QKC token (*uint256.Int)        │
│            MntBalances = MNT tokens (*TokenBalances) │
│            Stored in StateAccount directly           │
│                                                      │
│  Encoding boundary: updateStateObject → trie         │
│    StateAccount (5 fields) →  QuarkChain (6 elem)    │
│                                                      │
│  P2P sync: with pyquarkchain/goquarkchain nodes      │
│  Requirement: identical trie root hash               │
└──────────────────────────────────────────────────────┘
```

**Core design principle**:
- `StateAccount` struct adds `MntBalances *types.TokenBalances` as 5th field
- `*uint256.Int` balance type stays in the EVM and state machine for QKC
- RLP encoding at trie update time produces QuarkChain 6-element format
- All balance operations for non-QKC tokens go through new `MntBalances` system
- MNT precompiles handle token transfers, minting, and balance queries

---

## 4. MNT Token Model

### 4.1 Token Types

| Token Type | Internal Storage | RLP Storage | Example |
|-----------|-----------------|-------------|---------|
| QKC (native) | `StateAccount.Balance` (`*uint256.Int`) | Merged into `TokenBalances` as tokenID=35760 at encode time | Default token |
| Non-QKC MNT | `StateAccount.MntBalances` (`*TokenBalances`) | Merged into `TokenBalances` field in 6-element RLP | QKCUP, etc. |

**Key constraint**: QKC tokenID = `TokenIDEncode("QKC")` = 35760. All MNT balance operations (`SetMntBalance`, `AddMntBalance`, etc.) must **reject** calls with tokenID = QKC tokenID, since QKC is managed through `StateAccount.Balance`.

### 4.2 Token ID Encoding

36-base encoding: digits 0-9 (values 0-9), letters A-Z (values 10-35).
Max token name: 12 characters ("ZZZZZZZZZZZZ").

```
QKC  → tokenID = 35760  (0x8B98)
QKCUP → tokenID = computed via 36-base
```

**Encoding rule**: For each character from right to left:
```
id += charEncode(str[len-1])
for i from len-2 down to 0:
    id += base * (charEncode(str[i]) + 1)
    base *= 36
```

### 4.3 TokenBalances Data Structure

Two storage formats exist in goquarkchain based on number of non-zero token balances:

| Format | Condition | Encoding | goshard support |
|--------|-----------|----------|-----------------|
| List format | ≤ 16 non-zero balances | `0x00` prefix + RLP list of `TokenBalancePair` (sorted by TokenID asc) | **Implemented** |
| Trie format | > 16 non-zero balances | `0x01` prefix + 32 bytes SecureTrie merkle root | **Not implemented** |

**Trie format details (goquarkchain)**:
- Token IDs encoded as 32-byte keys, uint64 at bytes 24-31 (big-endian)
- Balances are RLP-encoded `*uint256.Int` (variable length)

**goshard limitation**: The trie format (`0x01`) is **not implemented**. `SerializeToBytes` returns an error if more than 16 non-zero token balances are present. `NewTokenBalancesFromBytes` returns an error on `0x01`-prefixed input.

**Reason**: Implementing the trie format requires a `SecureTrie` backed by a database. In `core/types`, importing the `trie` package creates a circular dependency (`core/types → trie → core/rawdb → consensus/misc/eip4844 → core/types`). A standalone MPT implementation exists in `core/types/mpt_hasher.go` for future use, but its output has not been verified byte-for-byte against goquarkchain's `SecureTrie`. No real mainnet accounts with > 16 MNT tokens have been observed.

---

## 5. Design Decisions

### 5.1 No `fullShardKey` on stateObject; MntBalances on StateAccount

**Rationale**: `FullShardKey` is per-shard, not per-account. Each shard uses a fixed shard key. The key can be derived from `Config` at encoding time based on the address's shard assignment.

`MntBalances` is now part of `StateAccount` (not a separate field on `stateObject`). This keeps all account data in one place and simplifies deepCopy/journal logic.

```go
// stateObject: no MNT-specific field needed — MntBalances lives in stateObject.data
type stateObject struct {
    // ... existing fields unchanged ...
    data        types.StateAccount  // Account data with all mutations applied in the scope of block
}
```

### 5.2 `GasTokenID` and `TransferTokenID` in EVM Context

Real mainnet data (`mainnet-mnt-txs.md`) shows transactions where `transfer_token_id` is non-QKC (e.g. 46347397) and `gas_token_id` is QKC (35760). These are plain EVM transactions — not `transferMnt` precompile calls — so the EVM `Context` must carry both fields exactly as goquarkchain does.

**goshard `Context` struct** (mirrors goquarkchain `core/vm/evm.go:95-96`):

```go
// core/vm/evm.go — Context struct
type Context struct {
    CanTransfer CanTransferFunc
    Transfer    TransferFunc
    GetHash     GetHashFunc

    Origin          common.Address
    GasPrice        *big.Int
    GasPriceInGasToken *big.Int
    Coinbase        common.Address
    GasLimit        uint64
    BlockNumber     *big.Int
    Time            *big.Int
    Difficulty      *big.Int
    GasTokenID      uint64   // tokenID used to pay gas (from tx.GasTokenID)
    TransferTokenID uint64   // tokenID used for value transfer (from tx.TransferTokenID)
}
```

**Function type signatures** (matching goquarkchain — tokenID is an explicit parameter):

```go
// core/evm.go
type CanTransferFunc func(StateDB, common.Address, *big.Int, uint64) bool
type TransferFunc    func(StateDB, common.Address, common.Address, *big.Int, uint64)
```

**`NewEVMContext`** sets both from the message (mirrors goquarkchain `core/evm.go:55-56`):

```go
func NewEVMContext(msg Message, header *types.Header, chain ChainContext) vm.Context {
    return vm.Context{
        CanTransfer:        CanTransfer,
        Transfer:           Transfer,
        // ...
        GasTokenID:         msg.GasTokenID(),
        TransferTokenID:    msg.TransferTokenID(),
    }
}
```

**`evm.Call()`** uses `evm.TransferTokenID` for balance checks and transfer (same as goquarkchain):

```go
if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value, evm.TransferTokenID) {
    return nil, gas, ErrInsufficientBalance
}
evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value, evm.TransferTokenID)
```

**`CanTransfer` / `Transfer`** implementations route on tokenID:

```go
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int, tokenID uint64) bool {
    if tokenID == defaultTokenID {
        return db.GetBalance(addr).Cmp(amount) >= 0
    }
    return db.GetMntBalance(addr, tokenID).Cmp((*uint256.Int)(amount)) >= 0
}

func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int, tokenID uint64) {
    if tokenID == defaultTokenID {
        db.SubBalance(sender, amount)
        db.AddBalance(recipient, amount)
    } else {
        amt := (*uint256.Int)(amount)
        db.SubMntBalance(sender, amt, tokenID)
        db.AddMntBalance(recipient, amt, tokenID)
    }
}
```

**`transferMnt` precompile** temporarily swaps `evm.TransferTokenID` around the inner `evm.Call`, then restores it — identical to goquarkchain:

```go
t := evm.TransferTokenID
evm.TransferTokenID = tokenID.Uint64()
ret, _, err := evm.Call(vm.AccountRef(caller), to, data, gas, value)
err = checkTokenIDQueried(err, contract, evm.TransferTokenID, defaultTokenID)
evm.TransferTokenID = t
```

**Gas payment** in `state_transition.go` uses `evm.GasTokenID` (mirrors goquarkchain `core/state_transition.go:170`):

```go
func (st *StateTransition) buyGas() error {
    mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
    if st.state.GetBalance(st.msg.From(), st.evm.GasTokenID).Cmp(mgval) < 0 {
        return errInsufficientBalanceForGas
    }
    st.state.SubBalance(st.msg.From(), mgval, st.evm.GasTokenID)
    // ...
}
```

**NOTE — Why `evm.TransferTokenID` cannot be removed:**

The `mnt-data.md` mainnet data contains two real transactions with `transfer_token_id = 46347397` (non-QKC, see `gas_token_is_qkc=true` / `transfer_token_is_qkc=false`). These are plain EVM value-transfer transactions — the token ID is set at the transaction level, not via the `transferMnt` precompile. If goshard tried to handle these by routing through `transferMnt` instead, the gas used would differ (at minimum +9000 for `CallValueTransferGas`), making the transaction receipt different and causing state root mismatch with goquarkchain. Therefore `evm.TransferTokenID` (and its routing through `CanTransfer`/`Transfer`) is a hard requirement for historical chain compatibility.

**QUESTION — Should `evm.GasTokenID` be retained?**

`mnt-data.md` shows `gas_token_id = 35760` (QKC) for both recorded transactions — i.e., gas is always paid in QKC in the observed data. goquarkchain supports non-QKC gas tokens but this involves a complex `PayNativeTokenAsGas` conversion path (non-default gas token → convert to genesis token price) that is not implemented in goshard. If all real mainnet transactions use QKC for gas, `evm.GasTokenID` could be simplified to always use `defaultTokenID`. However, it is retained for now to match goquarkchain's interface and avoid divergence if non-QKC gas transactions appear later.

### 5.3 Precompiles in `contracts_qkc.go`

The 5 MNT precompile implementations are ~400 lines. To keep `contracts.go` minimal, place them in a separate file. `contracts.go` only needs to merge `PrecompiledContractsMNT` into the active precompiles map.

### 5.4 `currentMntID` — `TokenIDQueried` and `checkTokenIDQueried`

`currentMntID` (`core/vm/contracts.go:534` in goquarkchain) returns `evm.TransferTokenID` and sets `contract.TokenIDQueried = true`. The `checkTokenIDQueried` logic lives **inside `evm.Call`**: after executing the recipient contract, `evm.Call` checks whether `contract.TokenIDQueried` was set — if not, and `evm.TransferTokenID != defaultTokenID` with `value > 0`, it reverts. `TokenIDQueried` is therefore a **contract-level** flag, read from the same `contract` local variable inside `evm.Call`.

**Both behaviors must be preserved in goshard** — removing `checkTokenIDQueried` would cause state divergence.

**Implementation in goshard** (identical to goquarkchain):

`TokenIDQueried` stays at the contract level:

```go
// core/vm/contract.go — add one field (mirrors goquarkchain)
type Contract struct {
    // ... existing fields unchanged ...
    TokenIDQueried bool  // set by currentMntID; checked by evm.Call after execution
}
```

`currentMntID` returns `evm.TransferTokenID` (the currently active MNT token ID, same as goquarkchain):

```go
func (c *currentMntID) Run(input []byte, evm *EVM, contract *Contract) ([]byte, error) {
    contract.TokenIDQueried = true
    output := make([]byte, 32)
    binary.BigEndian.PutUint64(output[24:], evm.TransferTokenID) // returns active transfer token ID
    return output, nil
}
```

`evm.Call` check condition uses `evm.TransferTokenID != defaultTokenID` (identical to goquarkchain):

```go
// Inside evm.Call, after run(evm, contract, input, false):
if err == nil && len(contract.Code) != 0 && !contract.TokenIDQueried &&
    evm.TransferTokenID != defaultTokenID && value.Sign() > 0 {
    err = ErrExecutionReverted
}
```

**Summary of changes vs goquarkchain**:

| Aspect | goquarkchain | goshard |
|--------|-------------|---------|
| `TokenIDQueried` location | `Contract` struct | `Contract` struct (same) |
| `GasTokenID` in `Context` | `uint64` | `uint64` (same) |
| `TransferTokenID` in `Context` | `uint64` | `uint64` (same) |
| Trigger condition in `evm.Call` | `evm.TransferTokenID != defaultTokenID` | `evm.TransferTokenID != defaultTokenID` (same) |
| `currentMntID` return value | `evm.TransferTokenID` (active MNT tokenID) | `evm.TransferTokenID` (same) |
| `CanTransferFunc` / `TransferFunc` signature | includes `uint64 tokenID` | includes `uint64 tokenID` (same) |

### 5.5 MNT Operations Must Reject QKC TokenID

All MNT balance operations must verify that the tokenID is NOT QKC's tokenID:

```go
const defaultTokenID = TokenIDEncode("QKC") // = 35760

func (s *stateObject) SetMntBalance(amount *uint256.Int, tokenID uint64) {
    if tokenID == defaultTokenID {
        panic("SetMntBalance called with QKC tokenID; use SetBalance instead")
    }
    // ...
}
```

This prevents accidental double-storage (QKC would exist in both `Balance` and `TokenBalances`).

---

## 6. File Plan

### 6.1 New Files (7 files)

| # | File Path | Description | Lines |
|---|-----------|-------------|-------|
| 1 | `core/types/token_balances.go` | `TokenBalances`, `TokenBalancePair` types | ~200 |
| 2 | `core/types/uint32_rlp.go` | `Uint32` custom RLP type for FullShardKey | ~50 |
| 3 | `common/token_codec.go` | Token ID 36-base encoding/decoding | ~80 |
| 4 | `common/utils.go` (add) | `EncodeToByte32` utility | ~5 |
| 5 | `core/state/state_object_qkc.go` | MNT accessors on stateObject | ~100 |
| 6 | `core/state/statedb_qkc.go` | StateDB MNT methods + QuarkChain encoder | ~180 |
| 7 | `core/vm/contracts_qkc.go` | 5 MNT precompile contracts | ~400 |

### 6.2 Modified Files (9 files)

| # | File Path | Change | Lines |
|---|-----------|--------|-------|
| 1 | `core/types/state_account.go` | Add `MntBalances` field; custom RLP encoder | ~25 |
| 2 | `core/types/transaction.go` | Add `GasTokenID`, `TransferTokenID` fields to transaction and `Message` interface | ~30 |
| 3 | `core/state/statedb.go` | MNT balance methods; modify `updateStateObject` | ~20 |
| 4 | `core/state/reader.go` | Decode QuarkChain 6-element format | ~40 |
| 5 | `core/state/journal.go` | MNT balance journal entries | ~30 |
| 6 | `core/vm/contracts.go` | Merge `PrecompiledContractsMNT` | ~10 |
| 7 | `core/vm/evm.go` | Add `GasTokenID`, `TransferTokenID` to `Context`; update `evm.Call()` to pass tokenID to `CanTransfer`/`Transfer`; update `checkTokenIDQueried` condition | ~20 |
| 8 | `core/evm.go` | `CanTransferFunc`/`TransferFunc` with `uint64 tokenID` param; `CanTransfer`/`Transfer` implementations routing on tokenID; `NewEVMContext` sets `GasTokenID`/`TransferTokenID` from message | ~40 |
| 9 | `core/state_transition.go` | `buyGas` uses `evm.GasTokenID`; `TransitionDb` uses `evm.TransferTokenID` for preCheck | ~30 |

**Total new code**: ~1035 lines | **Total modified code**: ~245 lines

---

## 7. Detailed Implementation

### 7.1 `core/types/token_balances.go` — TokenBalances

Ported from goquarkchain `core/types/token.go` with dependency path changes.

```go
package types

import (
    "bytes"
    "io"
    "sort"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/ethereum/go-ethereum/trie"
    "github.com/holiman/uint256"
)

const TokenTrieThreshold = 16

// TokenBalancePair is a single token-balance entry in the RLP list format.
type TokenBalancePair struct {
    TokenID uint64
    Balance *uint256.Int
}

// TokenBalances holds multiple token balances.
// When non-zero balances <= 16, stored as RLP list.
// When > 16, switches to a SecureTrie for efficient storage.
type TokenBalances struct {
    db        *trie.Database
    tokenTrie *trie.SecureTrie  // nil when using list format
    balances  map[uint64]*uint256.Int  // in-memory cache
}

// Constructors
func NewEmptyTokenBalances() *TokenBalances
func NewTokenBalancesWithMap(data map[uint64]*uint256.Int) *TokenBalances
func NewTokenBalances(data []byte, db *trie.Database) (*TokenBalances, error)

// Core operations
func (t *TokenBalances) SetValue(amount *uint256.Int, tokenID uint64)
func (t *TokenBalances) GetTokenBalance(tokenID uint64) *uint256.Int
func (t *TokenBalances) GetBalanceMap() map[uint64]*uint256.Int

// Helpers
func (t *TokenBalances) Len() int
func (t *TokenBalances) IsBlank() bool
func (t *TokenBalances) nonZeroEntriesInBalancesCache() int
func (t *TokenBalances) notUsingTrie() bool

// Commit flushes in-memory balances to the SecureTrie (if trie format).
func (t *TokenBalances) Commit(db *trie.Database)

// Serialization
func (t *TokenBalances) SerializeToBytes() ([]byte, error)
func (t *TokenBalances) EncodeRLP(w io.Writer) error
func (t *TokenBalances) DecodeRLP(s *rlp.Stream) error

// Copy
func (t *TokenBalances) Copy() *TokenBalances
```

**Key details**:
- `SerializeToBytes()` for list: `0x00` prefix + RLP-encoded `[]TokenBalancePair` (sorted by TokenID ascending, zero balances excluded)
- `SerializeToBytes()` for trie: `0x01` prefix + 32 bytes of SecureTrie merkle root
- Token IDs in trie are encoded as 32-byte keys (uint64 at bytes 24-31, big-endian)
- Balances in trie are RLP-encoded `*uint256.Int` (variable length)
- `Commit()` populates the SecureTrie from in-memory balances and computes root

### 7.2 `core/types/uint32_rlp.go` — Uint32

Ported from goquarkchain `core/types/special_rlp.go`.

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

func (u *Uint32) GetValue() uint32 {
    return uint32(*u)
}

func (u *Uint32) EncodeRLP(w io.Writer) error {
    bytes := make([]byte, rlpUint32Len)
    bytes[0] = rlpUint32Prefix
    binary.BigEndian.PutUint32(bytes[1:], uint32(*u))
    _, err := w.Write(bytes)
    return err
}

func (u *Uint32) DecodeRLP(s *rlp.Stream) error {
    data, err := s.Raw()
    if err != nil {
        return err
    }
    if len(data) != rlpUint32Len {
        return fmt.Errorf("len is %v should %v", len(data), rlpUint32Len)
    }
    if data[0] != rlpUint32Prefix {
        return fmt.Errorf("prefix is wrong, is %v should %v", data[0], rlpUint32Prefix)
    }
    *u = Uint32(binary.BigEndian.Uint32(data[1:]))
    return nil
}
```

**Encoding**: `0x84 YY YY YY YY` — `0x84` means "the following string is 4 bytes" in RLP (0x80 + 4); total encoding is 5 bytes (1 prefix + 4 data).

### 7.3 `common/token_codec.go` — Token ID Encoding

Ported from goquarkchain `common/token_codec.go`.

```go
package common

const (
    TokenBase  = uint64(36)
    TokenIDMax = uint64(4873763662273663091) // ZZZZZZZZZZZZ
)

func TokenIDEncode(str string) uint64
func TokenIdDecode(id uint64) (string, error)
func TokenCharEncode(char byte) uint64
func TokenCharDecode(id uint64) (byte, error)
func ReverseString(s string) string
```

### 7.4 `common/utils.go` — EncodeToByte32

Add to existing file:

```go
func EncodeToByte32(data uint64) []byte {
    ret := make([]byte, 32)
    binary.BigEndian.PutUint64(ret[24:], data)
    return ret
}
```

Encodes a uint64 token ID into a 32-byte key for the token trie.

### 7.5 `core/state/state_object_qkc.go` — MNT Extensions to stateObject

New file that adds MNT-specific methods to `stateObject`. All MNT data lives in `s.data.MntBalances` (`StateAccount.MntBalances`).

```go
package state

import (
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/log"
    "github.com/holiman/uint256"
)

const defaultTokenID = 35760 // TokenIDEncode("QKC")

// === MNT Balance Methods ===

// SetMntBalance sets the balance for a non-QKC token.
func (s *stateObject) SetMntBalance(amount *uint256.Int, tokenID uint64) {
    if tokenID == defaultTokenID {
        log.Error("SetMntBalance called with QKC tokenID; use SetBalance instead", "addr", s.address)
        return
    }
    if s.data.MntBalances == nil {
        s.data.MntBalances = types.NewEmptyTokenBalances()
    }
    s.data.MntBalances.SetValue(amount, tokenID)
}

// AddMntBalance adds to the balance of a non-QKC token.
func (s *stateObject) AddMntBalance(amount *uint256.Int, tokenID uint64) {
    if amount.IsZero() {
        return
    }
    if tokenID == defaultTokenID {
        log.Error("AddMntBalance called with QKC tokenID; use AddBalance instead", "addr", s.address)
        return
    }
    current := s.GetMntBalance(tokenID)
    s.SetMntBalance(new(uint256.Int).Add(current, amount), tokenID)
}

// SubMntBalance subtracts from the balance of a non-QKC token.
func (s *stateObject) SubMntBalance(amount *uint256.Int, tokenID uint64) {
    if amount.IsZero() {
        return
    }
    if tokenID == defaultTokenID {
        log.Error("SubMntBalance called with QKC tokenID; use SubBalance instead", "addr", s.address)
        return
    }
    current := s.GetMntBalance(tokenID)
    s.SetMntBalance(new(uint256.Int).Sub(current, amount), tokenID)
}

// GetMntBalance returns the balance of a non-QKC token.
func (s *stateObject) GetMntBalance(tokenID uint64) *uint256.Int {
    if s.data.MntBalances == nil {
        return new(uint256.Int)
    }
    return s.data.MntBalances.GetTokenBalance(tokenID)
}

// MntBalances returns the TokenBalances object (may be nil).
func (s *stateObject) MntBalances() *types.TokenBalances {
    return s.data.MntBalances
}

// IsBlankMnt returns true if there are no MNT tokens.
func (s *stateObject) IsBlankMnt() bool {
    return s.data.MntBalances == nil || s.data.MntBalances.IsBlank()
}
```

### 7.6 `core/state/statedb_qkc.go` — MNT Balance Methods + QuarkChain Encoding

New file that adds MNT-specific StateDB methods and the QuarkChain account encoder.

```go
package state

import (
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/holiman/uint256"
)

// ===== MNT Balance Methods on StateDB =====

// SetMntBalance sets the balance for a non-QKC token.
func (s *StateDB) SetMntBalance(addr common.Address, amount *uint256.Int, tokenID uint64) {
    obj := s.getOrNewStateObject(addr)
    if obj != nil {
        obj.SetMntBalance(amount, tokenID)
    }
}

// AddMntBalance adds to the balance of a non-QKC token.
func (s *StateDB) AddMntBalance(addr common.Address, amount *uint256.Int, tokenID uint64) {
    obj := s.getOrNewStateObject(addr)
    if obj != nil {
        obj.AddMntBalance(amount, tokenID)
    }
}

// SubMntBalance subtracts from the balance of a non-QKC token.
func (s *StateDB) SubMntBalance(addr common.Address, amount *uint256.Int, tokenID uint64) {
    obj := s.getOrNewStateObject(addr)
    if obj != nil {
        obj.SubMntBalance(amount, tokenID)
    }
}

// GetMntBalance returns the balance of a non-QKC token.
func (s *StateDB) GetMntBalance(addr common.Address, tokenID uint64) *uint256.Int {
    obj := s.getStateObject(addr)
    if obj == nil {
        return new(uint256.Int)
    }
    return obj.GetMntBalance(tokenID)
}

// ===== QuarkChain Account Encoder =====

// EncodeAccountRLP encodes a StateAccount into QuarkChain's 6-element RLP format.
//
// QuarkChain Account RLP: [Nonce, TokenBalances, Root, CodeHash, FullShardKey, Optional]
//
// The second element (TokenBalances) merges:
//   - acct.Balance (QKC native, tokenID=35760)
//   - acct.MntBalances (non-QKC MNT tokens)
//
// Parameters:
//   - acct: the goshard StateAccount
//   - fullShardKey: shard key for the address (derived from config)
//   - optional: additional bytes (typically nil)
func EncodeAccountRLP(acct *types.StateAccount, fullShardKey uint32, optional []byte) []byte {
    // Merge QKC balance (tokenID=35760) into a combined TokenBalances for RLP.
    merged := mergeTokenBalances(acct.Balance, acct.MntBalances)

    var items []interface{}
    items = append(items, acct.Nonce)

    if merged == nil || merged.IsBlank() {
        items = append(items, []byte(nil))
    } else {
        items = append(items, merged)
    }

    items = append(items, acct.Root[:])
    items = append(items, acct.CodeHash)

    shardKey := types.Uint32(fullShardKey)
    items = append(items, shardKey)
    items = append(items, optional)

    data, _ := rlp.EncodeToBytes(items)
    return data
}

// mergeTokenBalances returns a TokenBalances containing both the QKC native balance
// (as tokenID=defaultTokenID) and any existing MNT balances.
// Returns nil if both are zero/empty.
func mergeTokenBalances(qkcBalance *uint256.Int, mnt *types.TokenBalances) *types.TokenBalances {
    if (qkcBalance == nil || qkcBalance.IsZero()) && (mnt == nil || mnt.IsBlank()) {
        return nil
    }
    merged := types.NewEmptyTokenBalances()
    if mnt != nil {
        for id, bal := range mnt.GetBalanceMap() {
            merged.SetValue(bal, id)
        }
    }
    if qkcBalance != nil && !qkcBalance.IsZero() {
        merged.SetValue(qkcBalance, defaultTokenID) // defaultTokenID = 35760
    }
    return merged
}
```

### 7.7 Modified `core/types/state_account.go` — Add MntBalances Field

Add `MntBalances` as the 5th field. The standard `gen_account_rlp.go` is NOT used for trie storage — use `EncodeAccountRLP` instead. `deepCopy` in `stateObject` copies `data` (a `StateAccount` value), so `MntBalances` is automatically included.

```go
// core/types/state_account.go
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int             // 256-bit, QKC native token only
    Root     common.Hash              // 32 bytes, storage trie root
    CodeHash []byte                   // 32 bytes, emptyCodeHash if empty
    MntBalances *types.TokenBalances  // non-QKC MNT balances; nil = no MNT tokens
}
```

`deepCopy` in `stateObject` (around line 510) needs no change — it copies `s.data` by value and then does deep copies of pointer fields. Add `MntBalances` copy:

```go
func (s *stateObject) deepCopy(db *StateDB) *stateObject {
    obj := &stateObject{
        db:      db,
        address: s.address,
        data:    s.data,   // copies all 5 StateAccount fields including MntBalances pointer
        // ... other fields ...
    }
    if s.data.MntBalances != nil {
        obj.data.MntBalances = s.data.MntBalances.Copy()
    }
    // ... rest unchanged ...
    return obj
}
```

### 7.8 Modified `core/state/statedb.go` — updateStateObject + MNT Methods

The critical change: intercept RLP encoding to produce QuarkChain 6-element format.

```go
// Modified updateStateObject (around line 575):
func (s *StateDB) updateStateObject(obj *stateObject) {
    // === MNT: Encode as QuarkChain 6-element format directly into trie ===
    hk := crypto.Keccak256(obj.Address().Bytes())

    // FullShardKey is derived from config based on address, not stored per-object
    shardKey := s.getShardKeyForAddress(obj.Address())

    data := EncodeAccountRLP(
        &obj.data,   // StateAccount (5 fields: Nonce, Balance, Root, CodeHash, MntBalances)
        shardKey,    // FullShardKey from config
        nil,         // Optional
    )

    if err := s.trie.Update(hk, data); err != nil {
        s.setError(fmt.Errorf("updateStateObject (%x) error: %v", obj.Address(), err))
    }
    if obj.dirtyCode {
        s.trie.UpdateContractCode(obj.Address(), common.BytesToHash(obj.CodeHash()), obj.code)
    }
}

// MNT balance methods on StateDB are delegated to the stateObject methods (see statedb_qkc.go).

### 7.9 `core/vm/contracts_qkc.go` — MNT Precompiles

New file with 5 precompiled contracts using the `0x514b43` ("QKC") prefix scheme.

| # | Precompile | Address | Gas |
|---|-----------|---------|-----|
| 1 | `currentMntID` | `0x...0000514b430001` | 3 |
| 2 | `transferMnt` | `0x...0000514b430002` | dynamic |
| 3 | `deploySystemContract` | `0x...0000514b430003` | deployRootChainPoSWStakingContractGas |
| 4 | `mintMNT` | `0x...0000514b430004` | 9000 |
| 5 | `balanceMNT` | `0x...0000514b430005` | 400 |

#### 7.9.1 `currentMntID` — Query Current Token ID

**Address**: `0x000000000000000000000000000000514b430001`
**Gas**: 3
**Input**: None
**Output**: 32 bytes — first 24 bytes zero + last 8 bytes = QKC token ID (35760) big-endian uint64

Sets `contract.TokenIDQueried = true` (contract-level flag, checked by `evm.Call` after executing the recipient). Returns `evm.TransferTokenID` — the currently active transfer token ID (identical to goquarkchain).

```go
type currentMntID struct{}
func (c *currentMntID) RequiredGas(input []byte) uint64 { return 3 }
func (c *currentMntID) Run(input []byte, evm *EVM, contract *Contract) ([]byte, error) {
    contract.TokenIDQueried = true
    output := make([]byte, 32)
    binary.BigEndian.PutUint64(output[24:], evm.TransferTokenID) // returns active transfer token ID
    return output, nil
}
func (c *currentMntID) Name() string { return "currentMntID" }
```

#### 7.9.2 `transferMnt` — Transfer Token via Message Call

**Address**: `0x000000000000000000000000000000514b430002`
**Gas**: dynamic (base + CallValueTransferGas + CallNewAccountGas)
**Input**: 4 × 32 bytes

```
[offset 0]   to address (padded to 32 bytes)
[offset 32]  tokenID (uint256)
[offset 64]  value (uint256)
[offset 96]  data (bytes, optional)
```

**Flow**:
1. Parse 4 inputs from calldata
2. Reject if staticcall
3. Validate tokenID ≤ TOKENIDMAX
4. Validate tokenID ≠ QKC tokenID (QKC uses `AddBalance`/`SubBalance`)
5. Validate recipient ≠ transferMnt itself
6. Compute gas cost
7. Check caller has sufficient MNT balance: `GetMntBalance(caller, tokenID) >= value`
8. **Directly**: `SubMntBalance(caller, value, tokenID)` then `AddMntBalance(to, value, tokenID)`
9. **Directly**: `evm.Call(caller, to, data, gas, value)` (value is passed but EVM's CanTransfer now accepts tokenID)
10. Return inner call's return data

**Identical to goquarkchain**: Temporarily sets `evm.TransferTokenID = tokenID` around the inner `evm.Call`, then restores the original value. The `CanTransfer`/`Transfer` functions receive tokenID explicitly as a parameter, routing to MNT balance operations for non-QKC tokens.

#### 7.9.3 `deploySystemContract` — Deploy System Contract

**Address**: `0x000000000000000000000000000000514b430003`
**Gas**: `deployRootChainPoSWStakingContractGas`
**Input**: 1 × 32 bytes — contract index

| Index | Contract | Scope | Description |
|-------|---------|-------|-------------|
| 1 | ROOT_CHAIN_POSW | GLOBAL | POSW staking contract |
| 2 | NON_RESERVED_NATIVE_TOKEN | LOCAL_CHAIN_0 | Token manager |
| 3 | GENERAL_NATIVE_TOKEN | GLOBAL | General native token manager |

#### 7.9.4 `mintMNT` — Mint New Native Token

**Address**: `0x000000000000000000000000000000514b430004`
**Gas**: `CallValueTransferGas` (9000)
**Input**: 3 × 32 bytes

```
[offset 0]   minter address
[offset 32]  tokenID (uint256)
[offset 64]  amount (uint256)
```

**Restrictions**:
- Only callable by `NonReservedNativeTokenContract`
- Only allowed on chain ID 0
- Default token (QKC) cannot be minted
- Amount must be > 0

#### 7.9.5 `balanceMNT` — Query Token Balance

**Address**: `0x000000000000000000000000000000514b430005`
**Gas**: 400
**Input**: 2 × 32 bytes

```
[offset 0]   address (padded)
[offset 32]  tokenID (uint256)
```

**Output**: 32 bytes containing the balance

### 7.10 Precompile Registration

Add MNT precompiles to a new map and merge into active precompiles:

```go
// core/vm/contracts_qkc.go
var PrecompiledContractsMNT = PrecompiledContracts{
    common.HexToAddress("0x000000000000000000000000000000514b430001"): &currentMntID{},
    common.HexToAddress("0x000000000000000000000000000000514b430002"): &transferMnt{},
    common.HexToAddress("0x000000000000000000000000000000514b430003"): &deploySystemContract{},
    common.HexToAddress("0x000000000000000000000000000000514b430004"): &mintMNT{},
    common.HexToAddress("0x000000000000000000000000000000514b430005"): &balanceMNT{},
}

// In core/vm/contracts.go, activePrecompiledContracts():
func activePrecompiledContracts(rules params.Rules) PrecompiledContracts {
    base := // ... existing logic returns base precompiles ...
    // Merge MNT precompiles
    for addr, p := range PrecompiledContractsMNT {
        base[addr] = p
    }
    return base
}
```

### 7.11 System Contracts

System contracts are **regular contracts** (not precompiles) deployed at fixed addresses. Their bytecode is ported from goquarkchain and written into the state trie at genesis or on the first call to the `deploySystemContract` precompile. The `514b43` prefix sits at the **high end** of the address, clearly distinguishing them from precompile addresses where `514b43` sits at the low end:

| Address | Name | Scope | Purpose |
|---------|------|-------|---------|
| `0x514b430000000000000000000000000000000002` | `NonReservedNativeTokenContract` | LOCAL_CHAIN_0 | MNT token registration and mint authorization |
| `0x514b430000000000000000000000000000000003` | `GeneralNativeTokenContract` | GLOBAL | General native token management |

#### 7.11.1 NonReservedNativeTokenContract (`0x514b43...0002`)

**Purpose**: Any account can call this contract to register a new MNT token by paying a registration fee, gaining mint authorization.

```
Key interfaces (Solidity ABI from goquarkchain):

function registerToken(uint64 tokenID) external payable
    → registers tokenID as a valid MNT
    → records tokenID → owner mapping
    → enables subsequent mintMNT precompile calls

function mintToken(uint64 tokenID, address to, uint256 amount) external
    → only callable by the tokenID owner
    → internally calls mintMNT precompile (0x...514b430004)
    → mintMNT verifies caller == NonReservedNativeTokenContract (0x514b43...0002)
```

**Interaction with mintMNT precompile**:

```
User ──call──▶ NonReservedNativeTokenContract.mintToken(tokenID, to, amount)
                  │
                  └──call──▶ mintMNT precompile (0x000000000000000000000000000000514b430004)
                                  │ contract.CallerAddress == SystemContracts[NON_RESERVED_NATIVE_TOKEN].Address()  ✓
                                  │   (0x514b430000000000000000000000000000000002)
                                  │   on failure: contract.Gas = 0; return ErrInvalidSender
                                  │ tokenID != QKC (35760)                                                          ✓
                                  │ amount > 0                                                                      ✓
                                  └──▶ stateDB.AddMntBalance(to, amount, tokenID)
```

**Deployment**: deployed via the `deploySystemContract` precompile (index=2); bytecode is hardcoded in `contracts_qkc.go` (extracted from goquarkchain).

```go
const nonReservedNativeTokenBytecode = "0x..." // from goquarkchain

var nonReservedNativeTokenAddr = common.HexToAddress(
    "0x514b430000000000000000000000000000000002")
```

#### 7.11.2 GeneralNativeTokenContract (`0x514b43...0003`)

**Purpose**: Global native token management; provides unified cross-shard token information queries and management.

```
Key interfaces:

function getTokenBalance(address account, uint64 tokenID) external view returns (uint256)
    → delegates to balanceMNT precompile (0x...514b430005)

function isTokenRegistered(uint64 tokenID) external view returns (bool)
    → checks whether tokenID is registered in NonReservedNativeTokenContract
```

**Deployment**: deployed via the `deploySystemContract` precompile (index=3).

#### 7.11.3 deploySystemContract Full Implementation

```go
// core/vm/contracts_qkc.go

const (
    nonReservedNativeTokenAddr = "0x514b430000000000000000000000000000000002"
    generalNativeTokenAddr     = "0x514b430000000000000000000000000000000003"
)

func (c *deploySystemContract) Run(input []byte, evm *EVM) ([]byte, error) {
    if len(input) < 32 {
        return nil, ErrExecutionReverted
    }
    index := new(big.Int).SetBytes(input[:32]).Uint64()

    switch index {
    case 1: // ROOT_CHAIN_POSW — GLOBAL
        return deployContract(evm, rootChainPoSWBytecode, common.Address{})
    case 2: // NON_RESERVED_NATIVE_TOKEN — LOCAL_CHAIN_0
        addr := common.HexToAddress(nonReservedNativeTokenAddr)
        return deployContractToAddr(evm, nonReservedNativeTokenBytecode, addr)
    case 3: // GENERAL_NATIVE_TOKEN — GLOBAL
        addr := common.HexToAddress(generalNativeTokenAddr)
        return deployContractToAddr(evm, generalNativeTokenBytecode, addr)
    default:
        return nil, ErrExecutionReverted
    }
}
```

#### 7.11.4 mintMNT Caller Check Full Implementation

```go
// core/vm/contracts_qkc.go

// mintMNT.Run follows the goshard PrecompiledContractWithEVM interface;
// contract carries call context (CallerAddress, Gas).
func (c *mintMNT) Run(input []byte, evm *EVM, contract *Contract) ([]byte, error) {
    if evm.interpreter.readOnly {
        return nil, ErrWriteProtection
    }

    // === Caller check: must be NonReservedNativeTokenContract ===
    // Aligned with goquarkchain: use contract.CallerAddress (direct caller),
    // not evm.Origin (tx originator). On failure: zero Gas and return ErrInvalidSender.
    allowedSender := SystemContracts[NON_RESERVED_NATIVE_TOKEN].Address()
    if !bytes.Equal(allowedSender.Bytes(), contract.CallerAddress.Bytes()) {
        contract.Gas = 0
        return nil, ErrInvalidSender
    }

    if len(input) < 96 {
        return nil, ErrExecutionReverted
    }
    recipient := common.BytesToAddress(input[12:32])
    tokenID   := new(big.Int).SetBytes(input[32:64]).Uint64()
    amount    := new(uint256.Int).SetBytes32(input[64:96])

    if tokenID == defaultTokenID {
        contract.Gas = 0
        return nil, ErrInvalidSender // QKC cannot be minted via this precompile
    }
    if amount.IsZero() {
        return nil, ErrExecutionReverted
    }

    evm.StateDB.AddMntBalance(recipient, amount, tokenID)
    return nil, nil
}
```

#### 7.11.5 transferMnt Caller Check Full Implementation

`transferMnt` has no address-level caller restriction (any account or contract may call it), but enforces the following checks:

```go
func (c *transferMnt) Run(input []byte, evm *EVM, contract *Contract) ([]byte, error) {
    // 1. Reject staticcall
    if evm.interpreter.readOnly {
        return nil, ErrWriteProtection
    }
    if len(input) < 128 {
        return nil, ErrExecutionReverted
    }

    to      := common.BytesToAddress(input[12:32])
    tokenID := new(big.Int).SetBytes(input[32:64]).Uint64()
    value   := new(uint256.Int).SetBytes32(input[64:96])
    data    := input[96:]

    // 2. tokenID must be in valid range
    if tokenID > TokenIDMax {
        return nil, ErrExecutionReverted
    }
    // 3. QKC transfers must use AddBalance/SubBalance, not this precompile
    if tokenID == defaultTokenID {
        return nil, ErrExecutionReverted
    }
    // 4. Prevent recursive call: recipient cannot be transferMnt itself
    if to == transferMntAddr {
        return nil, ErrExecutionReverted
    }

    caller := contract.CallerAddress

    // 5. Caller must have sufficient MNT balance
    if evm.StateDB.GetMntBalance(caller, tokenID).Lt(value) {
        return nil, ErrExecutionReverted
    }

    // Directly adjust MNT balances before the inner Call
    evm.StateDB.SubMntBalance(caller, value, tokenID)
    evm.StateDB.AddMntBalance(to, value, tokenID)

    // 6. Temporarily set evm.TransferTokenID to the MNT token ID around the inner
    //    evm.Call, then restore — identical to goquarkchain's TransferTokenID swap.
    //    CanTransfer/Transfer already route on tokenID, so no function pointer swap needed.
    t := evm.TransferTokenID
    evm.TransferTokenID = tokenID

    ret, _, err := evm.Call(vm.AccountRef(caller), to, data, gas, value.ToBig())
    err = checkTokenIDQueried(err, contract, evm.TransferTokenID, defaultTokenID)

    evm.TransferTokenID = t

    return ret, err
}
```

### 7.13 `core/vm/evm.go` — GasTokenID, TransferTokenID + checkTokenIDQueried

Three changes (mirrors goquarkchain):

**1. Add `GasTokenID` and `TransferTokenID` to `Context`**:

```go
// core/vm/evm.go — Context struct
type Context struct {
    // ... existing fields ...
    GasTokenID      uint64  // tokenID used to pay gas (from tx.GasTokenID)
    TransferTokenID uint64  // tokenID used for value transfer (from tx.TransferTokenID)
}
```

**2. Update `evm.Call()` to pass `evm.TransferTokenID` to `CanTransfer`/`Transfer`** (mirrors goquarkchain line 208, 236):

```go
// core/vm/evm.go — inside Call()
if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value, evm.TransferTokenID) {
    return nil, gas, ErrInsufficientBalance
}
evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value, evm.TransferTokenID)
```

**3. Add `checkTokenIDQueried` check inside `evm.Call`** (after contract execution, mirrors goquarkchain line 256):

```go
// core/vm/evm.go — inside Call(), after run(evm, contract, input, false)
if err == nil && len(contract.Code) != 0 && !contract.TokenIDQueried &&
    evm.TransferTokenID != defaultTokenID && value.Sign() > 0 {
    ret = nil
    err = ErrExecutionReverted
}
```

`contract.TokenIDQueried` is the contract-level flag set by `currentMntID`. When `evm.TransferTokenID == defaultTokenID` (QKC), the check is never triggered.

**Also add `TokenIDQueried bool` to `Contract`** (`core/vm/contract.go`):

```go
type Contract struct {
    // ... existing fields unchanged ...
    TokenIDQueried bool  // set by currentMntID precompile during execution
}
```

### 7.14 Modified `core/state_transition.go` — MNT-Aware Transaction Flow

The state transition uses `evm.GasTokenID` for gas payment and `evm.TransferTokenID` for value transfer, mirroring goquarkchain.

**`buyGas`** deducts gas fee from `GasTokenID` balance (mirrors goquarkchain `state_transition.go:170`):

```go
func (st *StateTransition) buyGas() error {
    mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
    if st.state.GetBalance(st.msg.From(), st.evm.GasTokenID).Cmp(mgval) < 0 {
        return errInsufficientBalanceForGas
    }
    st.state.SubBalance(st.msg.From(), mgval, st.evm.GasTokenID)
    // ...
}
```

**`preCheck`** validates both token balances (mirrors goquarkchain `state_transition.go:325`):

```go
func (st *StateTransition) preCheck() error {
    // Check transfer token balance (may be non-QKC)
    if !st.evm.CanTransfer(st.state, st.msg.From(), st.value, st.evm.TransferTokenID) {
        return ErrInsufficientFunds
    }
    // ... nonce check ...
    return nil
}
```

For QKC-only transactions (`TransferTokenID == GasTokenID == defaultTokenID`), behavior is unchanged — `CanTransfer`/`Transfer` route to the standard `GetBalance`/`SubBalance`/`AddBalance` path.

### 7.15 Genesis Allocation

Genesis allocation supports MNT balances through a new field:

```go
// In params/config.go:
type GenesisAlloc struct {
    // ... existing fields unchanged ...
    // MNT: additional token balances for genesis accounts
    MNTBalances map[common.Address]map[uint64]*uint256.Int  // address → (tokenID → balance)
}
```

Genesis init populates `stateObject.tokenBalances` for each account with MNT balances.

---

## 8. RLP Encoding Byte-Level Comparison

### 8.1 goshard Standard (before MNT)

```
RLP: [nonce_u64, balance_256(32 bytes), root(32 bytes), codehash(32 bytes)]
     ^ 4-element list start
     | nonce: "0a" (for nonce=10)
     | balance: "a0" + 32 bytes (for non-zero balance)
     | root: "a0" + 32 bytes
     └ codehash: "a0" + 32 bytes
```

### 8.2 QuarkChain (after MNT)

```
RLP: [nonce_u64, TokenBalances_bytes, root(32 bytes), codehash(32 bytes), shardKey(5 bytes), optional]
     ^ 6-element list start
     | nonce: "0a" (for nonce=10)
     | TokenBalances:
     |   - nil → RLP empty string "80" (no MNT tokens)
     |   - list format: "c1" + 0x00 + RLP[TokenBalancePair...] (≤16 tokens)
     |   - trie format: "c1" + 0x01 + 32 bytes merkle root (>16 tokens)
     | root: "a0" + 32 bytes
     | codehash: "a0" + 32 bytes
     | shardKey: "84" + 4 bytes big-endian (e.g., "84 00 00 00 00" for key=0)
     └ optional: "80" (nil)
```

### 8.3 TokenBalances Serialization Formats

| Format | Bytes | Use Case |
|--------|-------|----------|
| nil | RLP empty string (0x80) | No TokenBalances field set |
| `0x00` + RLP list | Variable | ≤ 16 non-zero token balances |
| `0x01` + merkle root | 33 bytes | > 16 non-zero token balances |

### 8.4 Example: Account with QKCUP Token

```
Account at 0xAbC...123 in state trie:

Account RLP = [
  Nonce:         0,
  TokenBalances: 0x00 || RLP([
    {TokenID: QKCUP, Balance: 10^18},
  ]),
  Root:          0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421,
  CodeHash:      0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470,
  FullShardKey:  0x84 00 00 00 00,
  Optional:      nil,
]

TokenBalances encoding (1 balance, ≤ 16):
  0x00                    ← list format prefix
  c3                      ← RLP list of 1 element
    c4                    ← RLP list of 2 elements (TokenBalancePair)
      f8 20               ← RLP uint: QKCUP tokenID
      f8 21               ← RLP uint: 10^18
```

---

## 9. Balance Flow: QKC vs MNT

### 9.1 QKC (Native Token)

```
QKC balance → StateAccount.Balance (*uint256.Int)
              → Used internally by EVM and state machine
              → Standard geth balance operations:
                  AddBalance, SubBalance, GetBalance
              → transferMnt precompile does NOT handle QKC
              → At trie write time: merged into TokenBalances as tokenID=35760
                  (via mergeTokenBalances in EncodeAccountRLP)
```

### 9.2 MNT (Non-QKC Tokens)

```
MNT balance → StateAccount.MntBalances (*TokenBalances)
              → RLP: TokenBalances serialization (0x00 + list / 0x01 + root)
              → Encoded as element 2 of 6-element QuarkChain trie RLP
              → Operations:
                  AddMntBalance, SubMntBalance, GetMntBalance, SetMntBalance
              → Accessed via:
                  - transferMnt precompile (user transfers MNT)
                  - mintMNT precompile (mint new tokens)
                  - balanceMNT precompile (query balance)
              → MNT operations reject QKC tokenID:
                  if tokenID == defaultTokenID (35760) → log error, skip
```

### 9.3 How `transferMnt` Works (Identical to goquarkchain)

```
transferMnt precompile:
  Input: to, tokenID, value, data

  1. Check: GetMntBalance(caller, tokenID) >= value
  2. Directly: SubMntBalance(caller, value, tokenID)
  3. Directly: AddMntBalance(to, value, tokenID)
  4. Set: t = evm.TransferTokenID; evm.TransferTokenID = tokenID
  5. Call: evm.Call(caller, to, data, gas, value)
     → CanTransfer(StateDB, caller, value, evm.TransferTokenID) → routes to MNT balance
     → Transfer(StateDB, caller, to, value, evm.TransferTokenID) → routes to MNT balance
     → checkTokenIDQueried: if recipient contract didn't call currentMntID → revert
  6. Restore: evm.TransferTokenID = t
  7. Return inner call's return data
```

---

## 10. What Does NOT Change

- `*uint256.Int` balance type in EVM and state machine (QKC always uses `Balance`)
- Any EVM bytecode interpretation
- Snapshot tree logic
- Transaction signing/verification (new fields added but format is backward-compatible)
- P2P protocol
- State account decoding (stays 4-field format for internal use)
- Trie node structure or hash algorithm (still keccak256(RLP))
- `currentMntID` TokenIDQueried enforcement mechanism (identical to goquarkchain)

---

## 11. Implementation Order

### Phase 1: Core Types
1. `common/token_codec.go` — Token ID encoding
2. `common/utils.go` — EncodeToByte32
3. `core/types/uint32_rlp.go` — Uint32 custom RLP
4. `core/types/token_balances.go` — TokenBalances type

### Phase 2: State Extensions
5. `core/types/state_account.go` — Add `MntBalances` field to StateAccount
6. `core/state/state_object_qkc.go` — MNT accessor methods (use `s.data.MntBalances`)
7. `core/state/statedb_qkc.go` — MNT balance methods + `EncodeAccountRLP`
8. `core/state/statedb.go` — Modify `updateStateObject`, add MNT methods
9. `core/state/reader.go` — Decode QuarkChain 6-element format into `StateAccount.MntBalances`
10. `core/state/journal.go` — MNT balance journal entries

### Phase 3: EVM Integration
11. `core/evm.go` — `CanTransfer`/`Transfer` + `tokenID` param
12. `core/vm/contracts_qkc.go` — 5 MNT precompile contracts
13. `core/vm/contracts.go` — Merge `PrecompiledContractsMNT`
14. `core/state_transition.go` — MNT-aware preCheck (QKC path only)

### Phase 4: Genesis & Testing
15. Genesis alloc support for MNT balances
16. Unit tests: RLP byte-for-byte comparison with goquarkchain
17. Integration tests: Trie root hash consistency
18. End-to-end: Create MNT, transfer MNT, mint MNT

---

## 12. Testing Strategy

Three progressive test layers: **unit tests** (encoding correctness) → **snapshot replay** (static trie compatibility) → **EVM integration tests** (dynamic transaction compatibility).

---

### 12.1 Unit Tests

#### 12.1.1 RLP Byte-for-Byte Comparison

The most fundamental check: `EncodeAccountRLP` output for various account configurations must match golden bytes pre-generated by a goquarkchain Python script or Go test, byte-for-byte.

**Test case matrix**:

| Case | Balance(QKC) | MntBalances | FullShardKey | Notes |
|------|-------------|-------------|--------------|-------|
| 1 | 1000000 | nil | 0 | QKC-only account |
| 2 | 0 | {QKCUP: 500000} | 0 | MNT-only account |
| 3 | 1000000 | {QKCUP: 500000} | 0 | QKC + MNT |
| 4 | 1000000 | nil | 0x10000 | non-zero FullShardKey |
| 5 | 0 | 17 distinct tokens | 0 | exceeds threshold → trie format |

```go
func TestEncodeAccountRLP_MatchesGoQuarkchain(t *testing.T) {
    cases := []struct {
        name        string
        balance     *uint256.Int
        mntBalances map[uint64]*uint256.Int
        shardKey    uint32
        goldenHex   string // pre-generated from goquarkchain
    }{
        {"qkc_only",        uint256.NewInt(1000000), nil,                          0,       "..."},
        {"mnt_only",        uint256.NewInt(0),        map[uint64]*uint256.Int{TokenIDEncode("QKCUP"): uint256.NewInt(500000)}, 0, "..."},
        {"qkc_and_mnt",     uint256.NewInt(1000000),  map[uint64]*uint256.Int{TokenIDEncode("QKCUP"): uint256.NewInt(500000)}, 0, "..."},
        {"nonzero_shardkey",uint256.NewInt(1000000),  nil,                          0x10000, "..."},
        {"trie_format",     uint256.NewInt(0),         make17Tokens(),               0,       "..."},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            acct := &types.StateAccount{
                Nonce:       1,
                Balance:     c.balance,
                Root:        emptyRoot,
                CodeHash:    emptyCodeHash,
                MntBalances: types.NewTokenBalancesWithMap(c.mntBalances),
            }
            got := EncodeAccountRLP(acct, c.shardKey, nil)
            require.Equal(t, common.Hex2Bytes(c.goldenHex), got)
        })
    }
}
```

**Golden bytes generation** (goquarkchain Go test, `core/state/gen_golden_rlp_test.go`).
If the source data is in a pyquarkchain LevelDB, use the Python backup instead: `goshard/tools/generatedata/gen_golden_rlp.py`.

```go
// Run with: go test -run TestGenGoldenRLP -v
// Copy the printed hex into goshard's test cases.
func TestGenGoldenRLP(t *testing.T) {
    qkcupID := qkcCommon.TokenIDEncode("QKCUP")
    emptyRoot := types.EmptyRootHash
    emptyCode := crypto.Keccak256(nil)

    cases := []struct {
        name    string
        account *Account
        shardKey uint32
    }{
        {"qkc_only", &Account{
            Nonce:         1,
            TokenBalances: types.NewTokenBalancesWithMap(map[uint64]*big.Int{35760: big.NewInt(1000000)}),
            Root:          emptyRoot,
            CodeHash:      emptyCode,
            FullShardKey:  (*types.Uint32)(func() *uint32 { v := uint32(0); return &v }()),
        }, 0},
        {"mnt_only", &Account{
            Nonce:         1,
            TokenBalances: types.NewTokenBalancesWithMap(map[uint64]*big.Int{qkcupID: big.NewInt(500000)}),
            Root:          emptyRoot,
            CodeHash:      emptyCode,
            FullShardKey:  (*types.Uint32)(func() *uint32 { v := uint32(0); return &v }()),
        }, 0},
        {"qkc_and_mnt", &Account{
            Nonce: 1,
            TokenBalances: types.NewTokenBalancesWithMap(map[uint64]*big.Int{
                35760:   big.NewInt(1000000),
                qkcupID: big.NewInt(500000),
            }),
            Root:         emptyRoot,
            CodeHash:     emptyCode,
            FullShardKey: (*types.Uint32)(func() *uint32 { v := uint32(0); return &v }()),
        }, 0},
        {"nonzero_shardkey", &Account{
            Nonce:         1,
            TokenBalances: types.NewTokenBalancesWithMap(map[uint64]*big.Int{35760: big.NewInt(1000000)}),
            Root:          emptyRoot,
            CodeHash:      emptyCode,
            FullShardKey:  (*types.Uint32)(func() *uint32 { v := uint32(0x10000); return &v }()),
        }, 0x10000},
    }
    for _, c := range cases {
        encoded, err := rlp.EncodeToBytes(c.account)
        require.NoError(t, err)
        t.Logf("%s: %x", c.name, encoded)
    }
}
```

#### 12.1.2 MNT Rejects QKC TokenID

```go
func TestMntBalanceRejectsQKC(t *testing.T) {
    db, _ := state.New(common.Hash{}, backend)
    addr := common.HexToAddress("0x1234")
    db.CreateAccount(addr)

    // SetMntBalance with QKC tokenID must be a no-op (logs error, does not panic)
    db.SetMntBalance(addr, uint256.NewInt(100), 35760)
    require.Equal(t, uint256.NewInt(0), db.GetMntBalance(addr, 35760))
    require.Equal(t, uint256.NewInt(0), db.GetBalance(addr)) // QKC unaffected
}
```

#### 12.1.3 mergeTokenBalances Correctness

```go
func TestMergeTokenBalances(t *testing.T) {
    qkc := uint256.NewInt(1e18)
    mnt := types.NewTokenBalancesWithMap(map[uint64]*uint256.Int{
        TokenIDEncode("QKCUP"): uint256.NewInt(500),
    })
    merged := mergeTokenBalances(qkc, mnt)
    m := merged.GetBalanceMap()
    require.Equal(t, uint256.NewInt(1e18), m[35760])           // QKC
    require.Equal(t, uint256.NewInt(500),  m[TokenIDEncode("QKCUP")]) // MNT
}
```

---

### 12.2 Trie Snapshot Replay Test (Static Compatibility)

**Core idea**: export a real state trie from a running goquarkchain node (LevelDB snapshot), reload it using goshard code, re-encode every account, and verify that the reconstructed trie root hash exactly matches the original.

#### 12.2.1 Export goquarkchain Trie Snapshot

Go program in goquarkchain (`cmd/export_snapshot/main.go`), run against a live LevelDB.
If the source DB is from pyquarkchain, use the Python backup instead: `goshard/tools/generatedata/export_trie_snapshot.py`.

```go
// go run ./cmd/export_snapshot --db /path/to/shard-1.db --shard 1 --out testdata/trie_snapshot.json
package main

import (
    "encoding/json"
    "flag"
    "os"

    "github.com/QuarkChain/goquarkchain/core/state"
    "github.com/QuarkChain/goquarkchain/core/rawdb"
    qkcCommon "github.com/QuarkChain/goquarkchain/common"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rlp"
)

type accountEntry struct {
    Address     string            `json:"address"`
    RLPBytes    string            `json:"rlp_bytes"`
    Nonce       uint64            `json:"nonce"`
    QKCBalance  string            `json:"qkc_balance"`
    MntBalances map[string]string `json:"mnt_balances"`
    ShardKey    uint32            `json:"shard_key"`
}

type snapshot struct {
    RootHash string         `json:"root_hash"`
    Accounts []accountEntry `json:"accounts"`
}

func main() {
    dbPath := flag.String("db", "", "LevelDB path")
    out    := flag.String("out", "trie_snapshot.json", "output JSON path")
    flag.Parse()

    db, _ := rawdb.NewLevelDBDatabase(*dbPath, 128, 1024, "")
    stateDB, _ := state.New(readTipStateRoot(db), state.NewDatabase(db))

    snap := snapshot{RootHash: stateDB.IntermediateRoot(false).Hex()}

    stateDB.ForEachStorage(common.Address{}, func(addr common.Address, acct state.Account) bool {
        raw, _ := rlp.EncodeToBytes(acct)
        balMap := acct.TokenBalances.GetBalanceMap()
        mnt := make(map[string]string)
        for id, bal := range balMap {
            if id != qkcCommon.TokenIDEncode("QKC") {
                mnt[fmt.Sprintf("%d", id)] = bal.String()
            }
        }
        snap.Accounts = append(snap.Accounts, accountEntry{
            Address:    addr.Hex(),
            RLPBytes:   common.Bytes2Hex(raw),
            Nonce:      acct.Nonce,
            QKCBalance: balMap[qkcCommon.TokenIDEncode("QKC")].String(),
            MntBalances: mnt,
            ShardKey:   uint32(*acct.FullShardKey),
        })
        return true
    })

    f, _ := os.Create(*out)
    json.NewEncoder(f).Encode(snap)
}
```

#### 12.2.2 goshard Load and Verify

```go
// core/state/testdata/snapshot_test.go
func TestTrieSnapshotReplay(t *testing.T) {
    snapshot := loadSnapshot(t, "testdata/trie_snapshot.json")

    // 1. Populate an empty StateDB with every account from the snapshot
    trieDb := triedb.NewHashDB(rawdb.NewMemoryDatabase())
    stateDB, _ := state.New(common.Hash{}, trieDb)

    for _, acc := range snapshot.Accounts {
        addr := common.HexToAddress(acc.Address)
        stateDB.CreateAccount(addr)

        qkcBal, _ := new(uint256.Int).SetFromDecimal(acc.QKCBalance)
        stateDB.SetBalance(addr, qkcBal, tracing.BalanceChangeUnspecified)

        for tokenIDStr, balStr := range acc.MntBalances {
            tokenID, _ := strconv.ParseUint(tokenIDStr, 10, 64)
            bal, _ := new(uint256.Int).SetFromDecimal(balStr)
            stateDB.SetMntBalance(addr, bal, tokenID)
        }
        stateDB.SetNonce(addr, acc.Nonce)
    }

    // 2. Commit and retrieve root
    root, _ := stateDB.Commit(0, false)

    // 3. Compare against the original goquarkchain root
    expected := common.HexToHash(snapshot.RootHash)
    require.Equal(t, expected, root,
        "state trie root mismatch: goshard encoding differs from goquarkchain")
}
```

**What this validates**:
- QKC-only accounts encode correctly (`Balance` merged as tokenID=35760)
- FullShardKey encodes correctly (`0x84` + 4 bytes)
- MNT list-format accounts (≤16 tokens) match
- MNT trie-format accounts (>16 tokens) produce the same merkle root

---

### 12.3 EVM Integration Tests (Dynamic Transaction Compatibility)

Execute transactions on top of the trie built in 12.2 and verify that the resulting state root matches the root produced by goquarkchain after the same transaction sequence.

#### 12.3.1 QKC Transfer

```go
func TestEVM_QKCTransfer_MatchesGoQuarkchain(t *testing.T) {
    stateDB := loadSnapshotStateDB(t) // state from 12.2 snapshot

    sender   := common.HexToAddress("0xAlice")
    receiver := common.HexToAddress("0xBob")
    amount   := uint256.NewInt(1e18) // 1 QKC

    // Record balance before execution
    preSender := stateDB.GetBalance(sender)

    // Execute standard QKC transfer (tokenID=defaultTokenID)
    evm := newTestEVM(stateDB)
    _, _, err := evm.Call(vm.AccountRef(sender), receiver, nil, 21000, amount.ToBig())
    require.NoError(t, err)

    // Verify balance changes
    require.Equal(t,
        new(uint256.Int).Sub(preSender, amount),
        stateDB.GetBalance(sender))
    require.Equal(t, amount, stateDB.GetBalance(receiver))

    // Commit and compare root
    root, _ := stateDB.Commit(0, false)
    require.Equal(t, goldenRootAfterQKCTransfer, root)
}
```

#### 12.3.2 Create MNT (mintMNT precompile)

```go
func TestEVM_CreateMNT_MatchesGoQuarkchain(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)

    mintMNTAddr := common.HexToAddress("0x000000000000000000000000000000514b430004")

    minter    := common.HexToAddress("0xNonReservedNativeTokenContract")
    recipient := common.HexToAddress("0xAlice")
    newTokenID := common.TokenIDEncode("MYTKN")
    mintAmount := uint256.NewInt(1_000_000)

    // Build mintMNT calldata: [recipient(32), tokenID(32), amount(32)]
    input := encodeMintMNTInput(recipient, newTokenID, mintAmount)

    evm := newTestEVM(stateDB)
    evm.Origin = minter
    _, _, err := evm.Call(vm.AccountRef(minter), mintMNTAddr, input, 9000, big.NewInt(0))
    require.NoError(t, err)

    // Verify recipient received MYTKN
    bal := stateDB.GetMntBalance(recipient, newTokenID)
    require.Equal(t, mintAmount, bal)

    // Commit and compare root against goquarkchain after the same mintMNT call
    root, _ := stateDB.Commit(0, false)
    require.Equal(t, goldenRootAfterCreateMNT, root)
}
```

#### 12.3.3 Transfer MNT (transferMnt precompile)

```go
func TestEVM_TransferMNT_MatchesGoQuarkchain(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)
    // Pre-condition: give Alice some QKCUP
    qkcupID := common.TokenIDEncode("QKCUP")
    stateDB.SetMntBalance(alice, uint256.NewInt(5000), qkcupID)

    transferMNTAddr := common.HexToAddress("0x000000000000000000000000000000514b430002")

    // Build transferMnt calldata: [to(32), tokenID(32), value(32), data(optional)]
    transferAmount := uint256.NewInt(1000)
    input := encodeTransferMNTInput(bob, qkcupID, transferAmount, nil)

    evm := newTestEVM(stateDB)
    _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
    require.NoError(t, err)

    // Verify balance changes
    require.Equal(t, uint256.NewInt(4000), stateDB.GetMntBalance(alice, qkcupID))
    require.Equal(t, uint256.NewInt(1000), stateDB.GetMntBalance(bob,   qkcupID))

    // QKC balance must be unaffected
    require.Equal(t, preAliceQKC, stateDB.GetBalance(alice))

    // Commit and compare root
    root, _ := stateDB.Commit(0, false)
    require.Equal(t, goldenRootAfterTransferMNT, root)
}
```

#### 12.3.4 Direct Precompile Unit Tests (Bypassing System Contracts)

Both precompiles can be tested independently **without deploying any system contract**:

- **`transferMnt`**: no address-level caller restriction — directly testable by nature
- **`mintMNT`**: has a `contract.CallerAddress` check, but `Run()` is a plain Go function; construct a `vm.Contract` with `CallerAddress` set to the `NonReservedNativeTokenContract` address to bypass the deployment step entirely

```go
// Helper: build a Contract with a specific caller address
func newContractWithCaller(caller common.Address, gas uint64) *vm.Contract {
    callerRef := vm.AccountRef(caller)
    selfRef   := vm.AccountRef(common.HexToAddress("0x000000000000000000000000000000514b430004"))
    contract  := vm.NewContract(callerRef, selfRef, big.NewInt(0), gas)
    contract.CallerAddress = caller
    return contract
}
```

**mintMNT direct unit tests**

```go
func TestPrecompile_MintMNT_Direct(t *testing.T) {
    stateDB := newEmptyStateDB(t)   // no snapshot needed — empty stateDB is sufficient
    evm     := newTestEVM(stateDB)
    precompile := &mintMNT{}

    authorizedCaller   := SystemContracts[NON_RESERVED_NATIVE_TOKEN].Address()
    unauthorizedCaller := common.HexToAddress("0xDeadBeef")
    alice   := common.HexToAddress("0xAlice")
    tokenID := common.TokenIDEncode("MYTKN")

    t.Run("authorized_caller_succeeds", func(t *testing.T) {
        input    := encodeMintMNTInput(alice, tokenID, uint256.NewInt(1000))
        contract := newContractWithCaller(authorizedCaller, 9000)
        _, err   := precompile.Run(input, evm, contract)
        require.NoError(t, err)
        require.Equal(t, uint256.NewInt(1000), stateDB.GetMntBalance(alice, tokenID))
        require.Equal(t, uint64(0), contract.Gas) // gas consumed
    })

    t.Run("unauthorized_caller_gas_zeroed", func(t *testing.T) {
        input    := encodeMintMNTInput(alice, tokenID, uint256.NewInt(1000))
        contract := newContractWithCaller(unauthorizedCaller, 9000)
        _, err   := precompile.Run(input, evm, contract)
        require.ErrorIs(t, err, vm.ErrInvalidSender)
        require.Equal(t, uint64(0), contract.Gas)                                  // contract.Gas = 0
        require.Equal(t, uint256.NewInt(0), stateDB.GetMntBalance(alice, tokenID)) // nothing minted
    })

    t.Run("qkc_tokenid_gas_zeroed", func(t *testing.T) {
        input    := encodeMintMNTInput(alice, 35760, uint256.NewInt(1000)) // QKC
        contract := newContractWithCaller(authorizedCaller, 9000)
        _, err   := precompile.Run(input, evm, contract)
        require.ErrorIs(t, err, vm.ErrInvalidSender)
        require.Equal(t, uint64(0), contract.Gas)
    })

    t.Run("zero_amount_reverts", func(t *testing.T) {
        input    := encodeMintMNTInput(alice, tokenID, uint256.NewInt(0))
        contract := newContractWithCaller(authorizedCaller, 9000)
        _, err   := precompile.Run(input, evm, contract)
        require.ErrorIs(t, err, vm.ErrExecutionReverted)
        require.NotEqual(t, uint64(0), contract.Gas) // gas NOT zeroed (non-authorization error)
    })
}
```

**transferMnt direct unit tests**

```go
func TestPrecompile_TransferMNT_Direct(t *testing.T) {
    stateDB := newEmptyStateDB(t)
    evm     := newTestEVM(stateDB)
    precompile := &transferMnt{}

    alice   := common.HexToAddress("0xAlice")
    bob     := common.HexToAddress("0xBob")
    qkcupID := common.TokenIDEncode("QKCUP")
    stateDB.SetMntBalance(alice, uint256.NewInt(500), qkcupID)

    // For transferMnt, contract.CallerAddress is the actual token sender
    newContract := func(caller common.Address) *vm.Contract {
        c := vm.NewContract(vm.AccountRef(caller),
            vm.AccountRef(common.HexToAddress("0x000000000000000000000000000000514b430002")),
            big.NewInt(0), 50000)
        c.CallerAddress = caller
        return c
    }

    t.Run("normal_transfer", func(t *testing.T) {
        input    := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(100), nil)
        _, err   := precompile.Run(input, evm, newContract(alice))
        require.NoError(t, err)
        require.Equal(t, uint256.NewInt(400), stateDB.GetMntBalance(alice, qkcupID))
        require.Equal(t, uint256.NewInt(100), stateDB.GetMntBalance(bob,   qkcupID))
    })

    t.Run("insufficient_balance", func(t *testing.T) {
        input  := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(9999), nil)
        _, err := precompile.Run(input, evm, newContract(alice))
        require.ErrorIs(t, err, vm.ErrExecutionReverted)
    })

    t.Run("qkc_tokenid_rejected", func(t *testing.T) {
        input  := encodeTransferMNTInput(bob, 35760, uint256.NewInt(1), nil)
        _, err := precompile.Run(input, evm, newContract(alice))
        require.ErrorIs(t, err, vm.ErrExecutionReverted)
    })

    t.Run("staticcall_rejected", func(t *testing.T) {
        evm2 := newReadOnlyTestEVM(stateDB) // readOnly = true
        input  := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(1), nil)
        _, err := precompile.Run(input, evm2, newContract(alice))
        require.ErrorIs(t, err, vm.ErrWriteProtection)
    })
}
```

**Division of responsibility with L3 EVM tests**:

| Test type | Method | What it validates |
|-----------|--------|-------------------|
| Direct precompile unit tests (this section) | Call `Run()` directly with a crafted `Contract` | Internal logic, error codes, Gas behavior of each precompile in isolation |
| L3 EVM integration tests (12.3.2/12.3.3) | Full `evm.Call()` path | Gas metering, journal/revert, state root consistency |
| L3 System contract tests (12.3.6) | Call via contract ABI | Contract→precompile call chain, correct caller address propagation |

---

#### 12.3.5 Caller Validation Tests (via EVM Call path)

**mintMNT (`0x000000000000000000000000000000514b430004`) — caller must be NonReservedNativeTokenContract**

```go
func TestEVM_MintMNT_CallerValidation(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)
    mintMNTAddr := common.HexToAddress("0x000000000000000000000000000000514b430004")
    authorizedCaller := common.HexToAddress("0x514b430000000000000000000000000000000002")
    unauthorizedCaller := common.HexToAddress("0xDeadBeef")

    tokenID := common.TokenIDEncode("MYTKN")
    input := encodeMintMNTInput(common.HexToAddress("0xAlice"), tokenID, uint256.NewInt(1000))

    t.Run("authorized_caller_succeeds", func(t *testing.T) {
        evm := newTestEVMWithCaller(stateDB, authorizedCaller)
        _, _, err := evm.Call(vm.AccountRef(authorizedCaller), mintMNTAddr, input, 9000, big.NewInt(0))
        require.NoError(t, err)
        require.Equal(t, uint256.NewInt(1000),
            stateDB.GetMntBalance(common.HexToAddress("0xAlice"), tokenID))
    })

    t.Run("unauthorized_caller_reverts", func(t *testing.T) {
        evm := newTestEVMWithCaller(stateDB, unauthorizedCaller)
        _, _, err := evm.Call(vm.AccountRef(unauthorizedCaller), mintMNTAddr, input, 9000, big.NewInt(0))
        require.ErrorIs(t, err, vm.ErrInvalidSender) // contract.Gas = 0, ErrInvalidSender
    })

    t.Run("mint_qkc_tokenid_reverts", func(t *testing.T) {
        badInput := encodeMintMNTInput(common.HexToAddress("0xAlice"), 35760, uint256.NewInt(1000))
        evm := newTestEVMWithCaller(stateDB, authorizedCaller)
        _, _, err := evm.Call(vm.AccountRef(authorizedCaller), mintMNTAddr, badInput, 9000, big.NewInt(0))
        require.ErrorIs(t, err, vm.ErrInvalidSender) // tokenID == QKC → Gas=0, ErrInvalidSender
    })

    t.Run("mint_zero_amount_reverts", func(t *testing.T) {
        zeroInput := encodeMintMNTInput(common.HexToAddress("0xAlice"), tokenID, uint256.NewInt(0))
        evm := newTestEVMWithCaller(stateDB, authorizedCaller)
        _, _, err := evm.Call(vm.AccountRef(authorizedCaller), mintMNTAddr, zeroInput, 9000, big.NewInt(0))
        require.Error(t, err) // amount must be > 0
    })
}
```

**transferMnt (`0x000000000000000000000000000000514b430002`) — no address-level caller restriction, but enforces balance / staticcall / tokenID checks**

```go
func TestEVM_TransferMNT_CallerValidation(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)
    transferMNTAddr := common.HexToAddress("0x000000000000000000000000000000514b430002")
    alice := common.HexToAddress("0xAlice")
    bob   := common.HexToAddress("0xBob")
    qkcupID := common.TokenIDEncode("QKCUP")
    stateDB.SetMntBalance(alice, uint256.NewInt(500), qkcupID)

    t.Run("staticcall_reverts", func(t *testing.T) {
        evm := newTestEVM(stateDB)
        evm.interpreter.readOnly = true
        input := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(100), nil)
        _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
        require.Error(t, err) // ErrWriteProtection
    })

    t.Run("insufficient_mnt_balance_reverts", func(t *testing.T) {
        evm := newTestEVM(stateDB)
        input := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(99999), nil) // more than alice has
        _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
        require.Error(t, err)
    })

    t.Run("transfer_qkc_tokenid_reverts", func(t *testing.T) {
        evm := newTestEVM(stateDB)
        input := encodeTransferMNTInput(bob, 35760, uint256.NewInt(1), nil) // QKC via transferMnt
        _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
        require.Error(t, err) // use AddBalance/SubBalance for QKC
    })

    t.Run("transfer_to_self_precompile_reverts", func(t *testing.T) {
        evm := newTestEVM(stateDB)
        input := encodeTransferMNTInput(transferMNTAddr, qkcupID, uint256.NewInt(1), nil)
        _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
        require.Error(t, err) // to == transferMnt itself
    })

    t.Run("any_eoa_can_call", func(t *testing.T) {
        // No address restriction: any EOA may call transferMnt
        evm := newTestEVM(stateDB)
        input := encodeTransferMNTInput(bob, qkcupID, uint256.NewInt(100), nil)
        _, _, err := evm.Call(vm.AccountRef(alice), transferMNTAddr, input, 50000, big.NewInt(0))
        require.NoError(t, err)
        require.Equal(t, uint256.NewInt(400), stateDB.GetMntBalance(alice, qkcupID))
        require.Equal(t, uint256.NewInt(100), stateDB.GetMntBalance(bob,   qkcupID))
    })
}
```

#### 12.3.6 System Contract Tests

**deploySystemContract → NonReservedNativeTokenContract → mintMNT full call chain**

```go
func TestEVM_SystemContract_DeployAndMint(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)
    deployer         := common.HexToAddress("0xAdmin")
    deployPrecompile := common.HexToAddress("0x000000000000000000000000000000514b430003")
    nonReservedAddr  := common.HexToAddress("0x514b430000000000000000000000000000000002")

    // Step 1: Deploy NonReservedNativeTokenContract (index=2)
    t.Run("deploy_non_reserved_contract", func(t *testing.T) {
        input := encodeUint256(2)
        evm := newTestEVM(stateDB)
        _, _, err := evm.Call(vm.AccountRef(deployer), deployPrecompile, input, 5_000_000, big.NewInt(0))
        require.NoError(t, err)

        // Contract must have code at the fixed address
        require.NotEmpty(t, stateDB.GetCode(nonReservedAddr))
    })

    // Step 2: Call NonReservedNativeTokenContract.mintToken → mintMNT precompile
    t.Run("register_and_mint_new_token", func(t *testing.T) {
        newTokenID := common.TokenIDEncode("NEWCO")
        alice := common.HexToAddress("0xAlice")

        // mintToken internally calls mintMNT precompile with caller = nonReservedAddr ✓
        mintCalldata := encodeMintTokenABI(newTokenID, alice, uint256.NewInt(1000))
        evm := newTestEVM(stateDB)
        _, _, err := evm.Call(vm.AccountRef(deployer), nonReservedAddr, mintCalldata, 100_000, big.NewInt(0))
        require.NoError(t, err)

        require.Equal(t, uint256.NewInt(1000), stateDB.GetMntBalance(alice, newTokenID))
    })

    // Step 3: Commit and compare root against goquarkchain
    root, _ := stateDB.Commit(0, false)
    require.Equal(t, goldenRootAfterSystemContractMint, root)
}

func TestEVM_SystemContract_DeployGeneralNativeToken(t *testing.T) {
    stateDB := loadSnapshotStateDB(t)
    deployer          := common.HexToAddress("0xAdmin")
    deployPrecompile  := common.HexToAddress("0x000000000000000000000000000000514b430003")
    generalNativeAddr := common.HexToAddress("0x514b430000000000000000000000000000000003")

    // Deploy GeneralNativeTokenContract (index=3)
    input := encodeUint256(3)
    evm := newTestEVM(stateDB)
    _, _, err := evm.Call(vm.AccountRef(deployer), deployPrecompile, input, 5_000_000, big.NewInt(0))
    require.NoError(t, err)
    require.NotEmpty(t, stateDB.GetCode(generalNativeAddr))

    // Call getTokenBalance (view call) — delegates to balanceMNT precompile
    alice   := common.HexToAddress("0xAlice")
    qkcupID := common.TokenIDEncode("QKCUP")
    stateDB.SetMntBalance(alice, uint256.NewInt(777), qkcupID)

    queryCalldata := encodeGetTokenBalanceABI(alice, qkcupID)
    evm2 := newTestEVM(stateDB)
    ret, _, err := evm2.Call(vm.AccountRef(alice), generalNativeAddr, queryCalldata, 50_000, big.NewInt(0))
    require.NoError(t, err)

    got := new(uint256.Int).SetBytes(ret)
    require.Equal(t, uint256.NewInt(777), got)
}
```

#### 12.3.8 Golden Root Generation (goquarkchain side)

Go test in goquarkchain (`core/state/gen_golden_roots_test.go`), run once to print the expected roots.
If the source data is in a pyquarkchain LevelDB, use the Python backup instead: `goshard/tools/generatedata/gen_golden_roots.py`.

```go
// go test -run TestGenGoldenRoots -v
// Copy the printed hashes into goshard's goldenRoot* constants.
func TestGenGoldenRoots(t *testing.T) {
    snap := loadSnapshot(t, "testdata/trie_snapshot.json")
    stateDB := buildStateFromSnapshot(t, snap)
    evm     := newTestEVM(stateDB)

    alice := common.HexToAddress("0xAlice")
    bob   := common.HexToAddress("0xBob")

    // QKC transfer
    _, _, err := evm.Call(vm.AccountRef(alice), bob, nil, 21000, big.NewInt(1e18))
    require.NoError(t, err)
    root, _ := stateDB.Commit(0, false)
    t.Logf("after_qkc_transfer: %s", root.Hex())

    // Deploy NonReservedNativeTokenContract (index=2)
    deployAddr := common.HexToAddress("0x000000000000000000000000000000514b430003")
    _, _, err = evm.Call(vm.AccountRef(alice), deployAddr, encodeUint256(2), 5_000_000, big.NewInt(0))
    require.NoError(t, err)

    // Mint MYTKN via NonReservedNativeTokenContract
    nonReserved := common.HexToAddress("0x514b430000000000000000000000000000000002")
    myTKN := qkcCommon.TokenIDEncode("MYTKN")
    _, _, err = evm.Call(vm.AccountRef(alice), nonReserved,
        encodeMintTokenABI(myTKN, alice, big.NewInt(1_000_000)), 100_000, big.NewInt(0))
    require.NoError(t, err)
    root, _ = stateDB.Commit(0, false)
    t.Logf("after_create_mnt: %s", root.Hex())

    // Transfer QKCUP via transferMnt precompile
    transferMntAddr := common.HexToAddress("0x000000000000000000000000000000514b430002")
    qkcupID := qkcCommon.TokenIDEncode("QKCUP")
    stateDB.AddBalance(alice, big.NewInt(5000), qkcupID)
    _, _, err = evm.Call(vm.AccountRef(alice), transferMntAddr,
        encodeTransferMNTInput(bob, qkcupID, big.NewInt(1000), nil), 50_000, big.NewInt(0))
    require.NoError(t, err)
    root, _ = stateDB.Commit(0, false)
    t.Logf("after_transfer_mnt: %s", root.Hex())

    // Deploy GeneralNativeTokenContract (index=3)
    _, _, err = evm.Call(vm.AccountRef(alice), deployAddr, encodeUint256(3), 5_000_000, big.NewInt(0))
    require.NoError(t, err)
    root, _ = stateDB.Commit(0, false)
    t.Logf("after_deploy_general_native: %s", root.Hex())
}
```

---

### 12.4 Test Execution Order and Pass Criteria

| Layer | Test | Pass criteria | Dependencies |
|-------|------|---------------|-------------|
| L1 Unit | `TestEncodeAccountRLP` | byte-for-byte match against golden hex | golden generated by `TestGenGoldenRLP` in goquarkchain |
| L1 Unit | `TestMergeTokenBalances` | QKC and MNT fields merged correctly | none |
| L1 Unit | `TestMntBalanceRejectsQKC` | QKC balance unaffected | none |
| L1 Unit | `TestPrecompile_MintMNT_Direct` | caller/gas/error codes correct; **no snapshot needed** | none |
| L1 Unit | `TestPrecompile_TransferMNT_Direct` | balance/staticcall/QKC rejection correct; **no snapshot needed** | none |
| L2 Snapshot | `TestTrieSnapshotReplay` | root hash matches exactly | trie_snapshot.json exported |
| L3 EVM | `TestEVM_QKCTransfer` | root hash matches | L2 pass + golden roots |
| L3 EVM | `TestEVM_MintMNT_CallerValidation` | authorized succeeds, unauthorized reverts via evm.Call | L2 pass |
| L3 EVM | `TestEVM_TransferMNT_CallerValidation` | all checks enforced via evm.Call path | L2 pass |
| L3 EVM | `TestEVM_CreateMNT` | root hash matches | L2 pass + golden roots |
| L3 EVM | `TestEVM_TransferMNT` | root hash matches | L2 pass + golden roots |
| L3 EVM | `TestEVM_SystemContract_DeployAndMint` | contract→precompile chain + root hash | L2 pass + golden roots |
| L3 EVM | `TestEVM_SystemContract_DeployGeneralNativeToken` | deploy + view query correct | L2 pass |

**Address quick reference**:

| Type | Address format | Example |
|------|---------------|---------|
| Precompile | `0x0000...0000514b43XXXX` (`514b43` at low end) | `0x...514b430004` = mintMNT |
| System Contract | `0x514b430000...0000XXXX` (`514b43` at high end) | `0x514b43...0002` = NonReservedNativeToken |

**Must pass in order**: L1 failure → encoding/merge bug; L2 failure → trie construction or account decode issue; L3 Caller failure → precompile validation logic missing; L3 System Contract failure → contract deployment or precompile call chain issue.

---

## 13. Summary of Changes

| Category | File | Change Type | Lines |
|----------|------|-------------|-------|
| **New** | `common/token_codec.go` | Port from goquarkchain | ~80 |
| **New** | `common/utils.go` (add) | Add `EncodeToByte32` | ~5 |
| **New** | `core/types/uint32_rlp.go` | Port from goquarkchain | ~50 |
| **New** | `core/types/token_balances.go` | Port from goquarkchain | ~200 |
| **New** | `core/state/state_object_qkc.go` | MNT accessors (use `s.data.MntBalances`) | ~100 |
| **New** | `core/state/statedb_qkc.go` | MNT methods + encoder | ~180 |
| **New** | `core/vm/contracts_qkc.go` | 5 precompile contracts | ~400 |
| **Modified** | `core/types/state_account.go` | Add `MntBalances` field; custom RLP encoder | ~25 |
| **Modified** | `core/state/statedb.go` | MNT methods + modify updateStateObject | ~20 |
| **Modified** | `core/state/reader.go` | Decode QuarkChain 6-element format → `MntBalances` | ~40 |
| **Modified** | `core/state/journal.go` | MNT balance journal entries | ~30 |
| **Modified** | `core/vm/contracts.go` | Merge PrecompiledContractsMNT | ~10 |
| **Modified** | `core/types/transaction.go` | Add `GasTokenID`, `TransferTokenID` to transaction data and `Message` interface | ~30 |
| **Modified** | `core/vm/contracts_qkc.go` | `transferMnt`: swap `evm.TransferTokenID` around inner call (like goquarkchain) | ~10 |
| **Modified** | `core/vm/evm.go` | Add `GasTokenID`/`TransferTokenID` to `Context`; `Call()` passes tokenID to `CanTransfer`/`Transfer`; `checkTokenIDQueried` uses `TransferTokenID != defaultTokenID` | ~20 |
| **Modified** | `core/vm/contract.go` | Add `TokenIDQueried bool` to `Contract` struct | ~3 |
| **Modified** | `core/state_transition.go` | QKC preCheck only | ~30 |
| **Modified** | `params/config.go` | Genesis MNT alloc | ~15 |

**Total**: ~1200 lines of new/modified code

---

## 14. Journal Support for MNT

The journal tracks balance changes for snapshot/revert. Extend it for MNT. Since `MntBalances` now lives in `StateAccount` (part of `stateObject.data`), journal entries record the previous `MntBalances` state:

```go
// In journal.go, add journal entry type:
type mntBalanceChange struct {
    address common.Address
    prev    *types.TokenBalances  // entire MntBalances snapshot before change
}

func (ch mntBalanceChange) revert(s *StateDB) {
    obj := s.getStateObject(ch.address)
    if obj != nil {
        obj.data.MntBalances = ch.prev
    }
}

func (j *journal) mntBalanceChange(addr common.Address, prev *types.TokenBalances) {
    j.entries = append(j.entries, mntBalanceChange{addr, prev})
}
```

Before any MNT balance mutation in `stateObject`, snapshot the current `MntBalances` and append a journal entry so the change can be reverted during snapshot rollback.

---

## 15. Account Reading (Decode Path)

When reading from the trie, we receive QuarkChain 6-element format. We decode it directly into `StateAccount` (5 fields):

**Decoding flow**:

1. `reader.go` `Account()` method reads raw bytes from trie
2. Calls `DecodeAccountRLP(rawData)` → returns `*StateAccount` (with `MntBalances` populated)
3. `statedb.go` `newObject()` receives the full `StateAccount` as normal
4. No separate mechanism needed — `MntBalances` is part of the struct

The decode path requires modifying `reader.go` to recognize the 6-element RLP format vs 4-element format. If the RLP element count is 6, decode as QuarkChain format (populate `MntBalances`); if 4, decode as standard geth format (`MntBalances = nil`). This handles mixed-state scenarios.

---

## 16. Full Transaction Flow with MNT

### Scenario: Contract A calls `transferMnt` to send 100 tokens of tokenID=5 to Contract B

```
Contract A (via transferMnt precompile):
  Input: to=0xB, tokenID=5, value=100, data=[]
  │
  ▼
transferMnt.precompile.Run(input)
  │
  ├─ 1. Parse: to=0xB, tokenID=5, value=100
  ├─ 2. Check: GetMntBalance(0xA, 5) >= 100  → true
  ├─ 3. SubMntBalance(0xA, 100, 5)
  │     → MntBalances[5] = oldBalance - 100  [in-memory map]
  │     → journal: snapshot prev MntBalances
  ├─ 4. AddMntBalance(0xB, 100, 5)
  │     → MntBalances[5] = 100  [in-memory map, newly created]
  │     → journal: snapshot prev MntBalances (nil)
  ├─ 5. evm.Call(0xA, 0xB, data, gas, value)
  │     → CanTransfer(StateDB, 0xA, value, tokenID=5) → true
  │     → Transfer(StateDB, 0xA, 0xB, value, tokenID=5) → MNT balance ops
  └─ 6. Return inner call's return data
  │
  ▼
StateDB.Commit()
  │
  ├─ updateStateObject(stateObject(0xA))
  │   → EncodeAccountRLP(&obj.data, shardKey, nil)
  │     (obj.data.MntBalances = {5: oldBalance-100})
  │   → trie.Update(keccak(0xA), encodedBytes)
  │
  ├─ updateStateObject(stateObject(0xB))
  │   → EncodeAccountRLP(&obj.data, shardKey, nil)
  │     (obj.data.MntBalances = {5: 100})
  │   → trie.Update(keccak(0xB), encodedBytes)
  │
  └─ trie.Commit() → new state root hash
```

### Scenario: Standard QKC Transfer (unchanged)

```
Transaction (from=0xA, to=0xB, value=1 ether, gas=21000)
  │
  ▼
StateTransition.TransitionDb()
  │ preCheck: CanTransfer(StateDB, 0xA, 1 ether, defaultTokenID) → true
  │ buyGas: deduct gas with QKC
  │
  ▼
evm.Call(sender, to(), data, gas, value=1 ether)
  │ CanTransfer(StateDB, 0xA, 1 ether, defaultTokenID) → true
  │ CreateAccount(0xB)
  │ Transfer(StateDB, 0xA, 0xB, 1 ether, defaultTokenID)
  │   → SubBalance(0xA, 1 ether)
  │   → AddBalance(0xB, 1 ether)
  │
  ▼
StateDB.Commit()
  │
  ├─ updateStateObject(stateObject(0xA))
  │   → EncodeAccountRLP(&obj.data, shardKey, nil)  // obj.data.MntBalances = nil
  │   → trie.Update(keccak(0xA), encodedBytes)
  │
  └─ trie.Commit() → new state root hash
```

---

## 17. Key Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| RLP encoding mismatch | State trie root hash differs from goquarkchain | Byte-for-byte comparison tests against goquarkchain output |
| TokenBalances trie format complexity | Incorrect serialization for >16 tokens | Extensive unit tests for both list and trie formats |
| EVM balance flow conflicts | QKC and MNT balance operations interfere | Strict separation: Balance for QKC, TokenBalances for MNT |
| Precompile integration | MNT precompiles don't correctly interact with EVM | Step-by-step integration testing |
| Journal/revert handling | MNT balance changes not properly reverted | Journal entries for MNT balance changes, tested with revert scenarios |
| Decode path mismatch | Old 4-element vs new 6-element RLP confusion | Detect element count during decode; handle both formats |
