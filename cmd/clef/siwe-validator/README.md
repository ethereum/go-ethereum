# SIWE Validator for Clef

This directory implements a minimal Sign-In with Ethereum (SIWE) message validator for Clef,  
designed to verify incoming EIP-4361 formatted messages before approving signing requests.

The validator checks critical fields including:

- **Domain**: Ensures the requested domain matches an expected domain (e.g., `localhost:3000`).
- **Ethereum Address**: Verifies that a valid `0x` prefixed address is provided.
- **URI**: Confirms the target URI matches the expected resource.
- **Version**: Verifies that the SIWE version is `1`.
- **ChainID**: Ensures the chain ID matches the intended network (e.g., `1` for Ethereum Mainnet).
- **Nonce**: Checks that a unique nonce is included to prevent replay attacks.
- **Issued At**: Ensures the issued timestamp follows the ISO 8601/RFC3339 format.

Unlike previous implementations relying on external libraries such as `spruceid/siwe-go`,  
this version introduces a **lightweight internal parser** that directly processes SIWE messages,  
eliminating external dependencies and improving maintainability.

---

## How It Works

Upon receiving a signing request, Clef will invoke the `siwe-validator` binary,  
passing the SIWE message via standard input (stdin).

The validator parses the message line-by-line and verifies mandatory fields according to EIP-4361 specifications.

If validation passes, Clef proceeds with the signing flow. Otherwise, signing is rejected.

### Manually Testing the Validator

You can manually simulate a Clef signing request by piping a SIWE message into `siwe-validator`.  
For example:

```bash
echo "localhost:3000 wants you to sign in with your Ethereum account:
0x32e0556aeC41a34C3002a264f4694193EBCf44F7

URI: https://localhost:3000
Version: 1
ChainID: 1
Nonce: 32891756
Issued At: 2025-04-26T12:00:00Z" | ./siwe-validator
```

If the message is valid, `siwe-validator` will exit silently with code `0`.  
If the message is invalid, an error message will be printed to `stderr`.

---

## Test Data

The `testdata/genmsg_test.go` file provides a minimal static SIWE message generator for manual testing purposes.

It outputs a standardized EIP-4361 formatted message, allowing developers to easily validate the `siwe-validator` behavior.

This file is intended for manual verification only and is not part of the production codebase or automated tests.

To manually generate and test a SIWE message:

```bash
cd cmd/clef/siwevalidator
go run testdata/genmsg_test.go | ./siwe-validator
```

---

## Notes

- The validator currently supports basic field validation only.
- Future improvements may include supporting optional fields like `Resources`, `Expiration Time`, and `Request ID`.
- This implementation follows the EIP-4361.

---
