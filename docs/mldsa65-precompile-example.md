# ML-DSA-65 Precompile Contract Integration

This note documents the Solidity-side calling convention for the ML-DSA-65 verification precompile introduced at address `0x0000000000000000000000000000000000000101`.

It is intended as an implementation example for smart wallets, account abstraction flows, and EIP companion documentation.

## What The Precompile Does

The precompile verifies an ML-DSA-65 signature and returns a single 32-byte word:

- `0x000...0001` when verification succeeds
- `0x000...0000` otherwise

It does not revert on invalid signatures.

## Call Interface

The precompile does not expose a Solidity ABI in the usual sense. Contracts call it via `staticcall` with raw bytes.

Input encoding:

```text
input = publicKey || signature || message
```

For ML-DSA-65, the fixed sizes are:

- public key: `1952` bytes
- signature: `3309` bytes
- message: variable length

Minimal adapter shape:

```solidity
library MLDSA65 {
    address internal constant PRECOMPILE = address(0x0101);
    uint256 internal constant PUBLIC_KEY_SIZE = 1952;
    uint256 internal constant SIGNATURE_SIZE = 3309;

    function verify(
        bytes memory publicKey,
        bytes memory signature,
        bytes memory message
    ) internal view returns (bool) {
        require(publicKey.length == PUBLIC_KEY_SIZE, "bad ML-DSA public key length");
        require(signature.length == SIGNATURE_SIZE, "bad ML-DSA signature length");

        bytes memory input = bytes.concat(publicKey, signature, message);
        (bool ok, bytes memory out) = PRECOMPILE.staticcall(input);
        if (!ok || out.length != 32) {
            return false;
        }
        uint256 word;
        assembly {
            word := mload(add(out, 0x20))
        }
        return word == 1;
    }
}
```

## Recommended Signing Payload

Applications should not verify signatures over arbitrary free-form messages. The signed payload should bind the authorization to a specific chain, contract, and action.

At minimum, include:

- chain ID
- contract address
- nonce
- target address
- ETH value
- calldata hash
- optional deadline

That avoids replay across chains, contracts, or previously authorized actions.

## Example Smart Wallet Pattern

The following pattern uses:

- a stored ML-DSA-65 public key
- per-wallet nonce replay protection
- a deterministic signed digest
- precompile-backed verification before executing a call

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

library MLDSA65 {
    address internal constant PRECOMPILE = address(0x0101);
    uint256 internal constant PUBLIC_KEY_SIZE = 1952;
    uint256 internal constant SIGNATURE_SIZE = 3309;

    function verify(
        bytes memory publicKey,
        bytes memory signature,
        bytes memory message
    ) internal view returns (bool) {
        require(publicKey.length == PUBLIC_KEY_SIZE, "bad ML-DSA public key length");
        require(signature.length == SIGNATURE_SIZE, "bad ML-DSA signature length");

        bytes memory input = bytes.concat(publicKey, signature, message);
        (bool ok, bytes memory out) = PRECOMPILE.staticcall(input);
        if (!ok || out.length != 32) {
            return false;
        }

        uint256 word;
        assembly {
            word := mload(add(out, 0x20))
        }
        return word == 1;
    }
}

contract PQSmartWallet {
    bytes public ownerPublicKey;
    uint256 public nonce;

    event Executed(address indexed target, uint256 value, bytes data, uint256 nonce);
    event PublicKeyRotated(bytes newPublicKey);

    constructor(bytes memory initialPublicKey) {
        require(initialPublicKey.length == MLDSA65.PUBLIC_KEY_SIZE, "bad ML-DSA public key length");
        ownerPublicKey = initialPublicKey;
    }

    function digestForExecute(
        address target,
        uint256 value,
        bytes calldata data,
        uint256 expectedNonce,
        uint256 deadline
    ) public view returns (bytes32) {
        return keccak256(
            abi.encode(
                bytes32("MLDSA65_WALLET_EXECUTE"),
                block.chainid,
                address(this),
                expectedNonce,
                deadline,
                target,
                value,
                keccak256(data)
            )
        );
    }

    function execute(
        address target,
        uint256 value,
        bytes calldata data,
        uint256 deadline,
        bytes calldata signature
    ) external payable {
        require(block.timestamp <= deadline, "signature expired");

        uint256 currentNonce = nonce;
        bytes32 digest = digestForExecute(target, value, data, currentNonce, deadline);
        bytes memory message = abi.encodePacked(digest);

        require(MLDSA65.verify(ownerPublicKey, signature, message), "invalid ML-DSA signature");

        nonce = currentNonce + 1;

        (bool ok, ) = target.call{value: value}(data);
        require(ok, "call failed");

        emit Executed(target, value, data, currentNonce);
    }

    function rotateKey(
        bytes calldata newPublicKey,
        uint256 deadline,
        bytes calldata signature
    ) external {
        require(newPublicKey.length == MLDSA65.PUBLIC_KEY_SIZE, "bad ML-DSA public key length");
        require(block.timestamp <= deadline, "signature expired");

        uint256 currentNonce = nonce;
        bytes32 digest = keccak256(
            abi.encode(
                bytes32("MLDSA65_WALLET_ROTATE"),
                block.chainid,
                address(this),
                currentNonce,
                deadline,
                keccak256(newPublicKey)
            )
        );

        require(
            MLDSA65.verify(ownerPublicKey, signature, abi.encodePacked(digest)),
            "invalid ML-DSA signature"
        );

        ownerPublicKey = newPublicKey;
        nonce = currentNonce + 1;

        emit PublicKeyRotated(newPublicKey);
    }
}
```

## Security Notes

This precompile only provides post-quantum verification as a building block. Whether a contract is quantum-hardened depends on the authorization policy around it.

Good pattern:

- contract state changes require a valid ML-DSA signature
- no ECDSA owner bypass exists
- key rotation is also ML-DSA authorized
- replay protection is present

Weak pattern:

- `msg.sender` remains the real authority
- ECDSA is accepted as a fallback
- signed messages omit chain ID, nonce, or contract binding

The security gain is in authorization of state transitions. This does not encrypt storage or make legacy EOAs quantum-safe.

## Reference Files

- Client precompile implementation: `core/vm/precompile_mldsa65.go`
- Precompile activation wiring: `params/config.go`
- Client-side tests: `core/vm/contracts_test.go`
