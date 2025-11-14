# Solidity

## Compatibility

- mainnet: v0.8.23
- testnet: v0.8.28
- devnet: v0.8.28

## Special variables

### block.prevrandao

The value of `block.prevrandao` is `keccak256(block.number)` in our current implemention. It is predictable and unsafe.

**NOTICE: do not use it in real business.**

### block.basefee

The value of `block.basefee` is always 12.5 GWei in our EIP-1559 implemention.

### block.blobbasefee

The value of `block.blobbasefee` is always 0 in our EIP-7516 implemention.
