# XDC Genesis Hash Compatibility Fix

**Date:** 2026-01-29  
**Status:** ✅ RESOLVED  

## Problem Statement

The Part4 implementation (go-ethereum v1.15 based XDC port) was producing different genesis hashes than the official XDPoSChain implementation when initializing from the XDC mainnet genesis file.

- **Expected Hash:** `0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1`
- **Part4 Produced:** `0x0683984f78dcbdd71f618cca7e14811b500b8f0f5ee1f1962d8fc44aa5a75eea` (before fix)

This would prevent Part4 nodes from syncing with the XDC mainnet.

---

## Root Causes Identified

### 1. Corrupted Genesis File

**Issue:** The Part4 genesis file (`genesis/xdc_mainnet.json`) had corrupted contract bytecode for the validator contract at address `0x88`.

| File | Code Length (bytes) | 
|------|-------------------|
| Part4 (corrupted) | 14,453 |
| Official | 14,452 |

The Part4 file had 2 extra hex characters (`00`) inserted at position 18745 in the bytecode.

**Impact:** Different code hash → different account hash → different state root.

**Fix:** Copied correct genesis from official repository and converted `xdc` address prefixes to `0x`.

### 2. Missing XDPoS Fields in RLP Encoder

**Issue:** The generated RLP encoder (`core/types/gen_header_rlp.go`) did not include the XDPoS-specific header fields.

XDC headers have three additional fields compared to standard Ethereum headers:
```go
Validators []byte  // List of validators
Validator  []byte  // Current validator
Penalties  []byte  // Penalty data
```

The generated code jumped from `Nonce` directly to `BaseFee`, skipping these fields entirely:

```go
// BEFORE (incorrect)
w.WriteBytes(obj.Nonce[:])
_tmp1 := obj.BaseFee != nil  // XDPoS fields skipped!
```

**Impact:** Header RLP encoding was 3 bytes shorter (661 vs 664 bytes), producing different block hashes.

**Fix:** Added XDPoS field encoding to `gen_header_rlp.go`:
```go
// AFTER (correct)
w.WriteBytes(obj.Nonce[:])
// XDPoS fields - always encode (not optional)
w.WriteBytes(obj.Validators)
w.WriteBytes(obj.Validator)
w.WriteBytes(obj.Penalties)
_tmp1 := obj.BaseFee != nil
```

### 3. Nil vs Empty Slice Initialization

**Issue:** Genesis block construction left XDPoS fields as `nil` instead of empty slices `[]byte{}`.

```go
// BEFORE (incorrect)
head := &types.Header{
    // ... other fields
    // Validators, Validator, Penalties not set (nil)
}
```

**Impact:** Even with proper RLP encoding, nil slices might be handled differently than empty slices.

**Fix:** Explicitly initialize to empty slices in `core/genesis.go`:
```go
// AFTER (correct)
head := &types.Header{
    // ... other fields
    Validators: []byte{},
    Validator:  []byte{},
    Penalties:  []byte{},
}
```

### 4. Incorrect Config Hash

**Issue:** `params/config.go` had wrong `XDCMainnetGenesisHash`.

```go
// BEFORE (incorrect)
XDCMainnetGenesisHash = common.HexToHash("0x81b02e6c24c0ed8383dd5f6c1e83e82b8f988af91f89f9b95c10dbd3e25cd025")
```

**Fix:**
```go
// AFTER (correct)
XDCMainnetGenesisHash = common.HexToHash("0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1")
```

---

## Files Modified

| File | Change |
|------|--------|
| `genesis/xdc_mainnet.json` | Replaced with correct genesis (fixed contract bytecode) |
| `core/types/gen_header_rlp.go` | Added XDPoS field encoding |
| `core/types/block.go` | Removed `rlp:"optional"` tags from XDPoS fields |
| `core/genesis.go` | Initialize XDPoS fields to empty slices |
| `params/config.go` | Fixed `XDCMainnetGenesisHash` value |

---

## Verification

### State Root Match
```
Part4:    0x49be235b0098b048f9805aed38a279d8c189b469ff9ba307b39c7ad3a3bc55ae
Official: 0x49be235b0098b048f9805aed38a279d8c189b469ff9ba307b39c7ad3a3bc55ae
✅ MATCH
```

### Header RLP Match
```
Part4:    664 bytes, ends with 808080 (three empty XDPoS fields)
Official: 664 bytes, ends with 808080
✅ MATCH
```

### Genesis Block Hash Match
```
Part4:    0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1
Official: 0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1
✅ MATCH
```

### Init Command
```bash
$ ./geth init --datadir /tmp/test genesis/xdc_mainnet.json
INFO Successfully wrote genesis state  database=chaindata hash=4a9d74..42d6b1
✅ SUCCESS
```

---

## Technical Details

### XDC Header Structure (RLP Order)

```
1.  ParentHash     [32]byte
2.  UncleHash      [32]byte
3.  Coinbase       [20]byte
4.  Root           [32]byte  (state root)
5.  TxHash         [32]byte
6.  ReceiptHash    [32]byte
7.  Bloom          [256]byte
8.  Difficulty     *big.Int
9.  Number         *big.Int
10. GasLimit       uint64
11. GasUsed        uint64
12. Time           uint64
13. Extra          []byte
14. MixDigest      [32]byte
15. Nonce          [8]byte
16. Validators     []byte    ← XDPoS field
17. Validator      []byte    ← XDPoS field
18. Penalties      []byte    ← XDPoS field
19. BaseFee        *big.Int  (optional, EIP-1559)
... (other optional fields)
```

### RLP Encoding for Empty XDPoS Fields

Empty `[]byte{}` encodes as `0x80` in RLP (single byte representing empty string).

Genesis block has all three XDPoS fields empty, so the header ends with:
```
...880000000000000000 808080
   └── Nonce (8 bytes) └── Validators, Validator, Penalties (empty)
```

### Why Generated Code Needed Manual Update

Go-ethereum uses code generation for RLP encoding:
```go
//go:generate go run ../../rlp/rlpgen -type Header -out gen_header_rlp.go
```

When new fields are added to the Header struct, the generated code must be regenerated or manually updated. In this case, manual update was safer to avoid unintended changes.

---

## Debugging Methodology

1. **Isolated the issue** - Verified trie encoding works correctly with simple test cases
2. **Binary comparison** - Compared RLP-encoded headers byte-by-byte
3. **Field-by-field verification** - Confirmed each header field matches
4. **Size discrepancy** - Found 3-byte difference (661 vs 664 bytes)
5. **Traced to source** - Found generated RLP encoder missing XDPoS fields
6. **Genesis file audit** - Found corrupted contract bytecode

---

## Lessons Learned

1. **Always verify genesis files** - Even small corruption (2 bytes) breaks everything
2. **Generated code needs updates** - When adding struct fields, check generated encoders
3. **nil vs empty slice matters** - In RLP encoding, they can behave differently
4. **Compare binary output** - When hashes don't match, compare the raw bytes

---

## References

- Official XDC genesis hash: `0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1`
- Official XDC state root: `0x49be235b0098b048f9805aed38a279d8c189b469ff9ba307b39c7ad3a3bc55ae`
- XDPoSChain repository: https://github.com/XinFinOrg/XDPoSChain
