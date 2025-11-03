<!-- Geth v1.16.6 Release Notes -->

This is a maintenance release with bug fixes, performance improvements, and several enhancements.

### All Changes

### Core

- Re-structured the trienode history header section and enabled partial freezer read for trienode data retrieval. (#32907)
- Fixed legacy chain freezer directory detection to only use legacy format when the directory is non-empty. (#33032)
- Fixed shared chainId modification between tests that could cause test failures. (#33020)
- Reduced memory allocations in `AccessList.Copy` and `modernSigner.Equal` for improved performance. (#33024, #32971)

### RPC & Tracing

- Fixed `prestateTracer` for EIP-6780 SELFDESTRUCT operations. (#33050)
- Fixed crasher in `TraceCall` when using `BlockOverrides`. (#33015)
- Removed unused error variables in RPC code. (#33012)

### Networking & P2P

- Ensured bootstrap completion before accepting incoming connections in discovery table to prevent test hangs. (#32881)
- Added cleanup of v4 discovery resources when v5 initialization fails. (#33005)
- Silenced spurious "Read error" log messages during listener shutdown. (#33001)

### CLI & Commands

- Added `--genesis` flag to set genesis configuration from file, providing an alternative to `geth init`. (#32844)
- Fixed maximum uint64 value expression in Era import (was incorrectly using bitwise XOR instead of max value). (#32934)
- Simplified address validation by using `IsHexAddress` method consistently. (#32997)
- Distinguished JWT secret handling between devp2p and geth, clarifying that `--authrpc.jwtsecret` expects a file path. (#32972)

### JavaScript Runtime

- Fixed `setTimeout`/`setInterval` callback argument forwarding to match standard JavaScript semantics. Only extra arguments (after callback and delay) are now passed to callbacks. (#32936)

### Testing & CI

- Fixed incorrect waitgroup usage in `XTestDelivery` test case. (#33047)
- Fixed error assertion in `accounts/abi/bind/v2` test. (#33041)
- Fixed flaky websocket test by adding minimum delay to prevent immediate connection resets. (#33002)
- Fixed flaky `TestSizeTracker` by ensuring pathdb buffer is fully flushed before baseline iteration. (#32993)
- Added 32-bit CI targets for keeper executables and unit tests. (#32911)
- Fixed keeper build in CI after workspace file removal. (#33018, #32632)

### Code Quality & Maintenance

- Simplified `FileExist` helper function for better readability. (#32969)
- Improved duration comparison in `PrettyAge` formatter to treat exact unit boundaries correctly. (#33064)
- Fixed `ChainConfig` logging to display actual timestamp values instead of pointer addresses. (#32766)
- Removed unused variables and improved code clarity. (#32989)

For a full rundown of the changes please consult the Geth 1.16.6 [release milestone](https://github.com/ethereum/go-ethereum/milestone/195?closed=1).

As with all our previous releases, you can find the:

- Pre-built binaries for all platforms on our [downloads page](https://geth.ethereum.org/downloads/).
- Docker images published under [`ethereum/client-go`](https://cloud.docker.com/u/ethereum/repository/docker/ethereum/client-go).
- Ubuntu packages in our [Launchpad PPA repository](https://launchpad.net/~ethereum/+archive/ubuntu/ethereum).
- OSX packages in our [Homebrew Tap repository](https://github.com/ethereum/homebrew-ethereum).
