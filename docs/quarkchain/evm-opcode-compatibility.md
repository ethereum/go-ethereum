# EVM Opcode Compatibility Differences: goquarkchain vs geth 1.17.3

## Background and replay risk

`goquarkchain` embeds an EVM derived from an old geth line. Its module file depends on `github.com/ethereum/go-ethereum v1.8.20`, and its default EVM chain config enables the pre-Istanbul fork set up to Constantinople at block zero. In contrast, `geth 1.17.3` contains fork-specific jump tables and EIP activators through later protocol changes, including Istanbul, Berlin, London, Merge, Shanghai, Cancun, Prague, Osaka, and Amsterdam-era logic.

This matters for historical replay. If historical `goquarkchain` blocks are executed under `geth 1.17.3` latest rules without a compatibility layer, the same bytecode can observe different opcode availability, different gas costs, different refunds, and different opcode semantics. Those differences can change transaction success, gasleft-dependent control flow, account/storage writes, and ultimately state roots.

Primary source anchors:

- `goquarkchain` default Constantinople-style config: [params/evm_params.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/params/evm_params.go#L36-L44)
- `goquarkchain` old geth dependency: [go.mod](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/go.mod#L1-L12)
- `goquarkchain` jump tables: [core/vm/jump_table.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/core/vm/jump_table.go#L53-L90)
- `geth 1.17.3` forked jump tables: [core/vm/jump_table.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/jump_table.go#L96-L170)
- `geth 1.17.3` opcode constants: [core/vm/opcodes.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/opcodes.go#L49-L125)
- `geth 1.17.3` EIP activators: [core/vm/eips.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/eips.go#L73-L190), [core/vm/eips.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/eips.go#L226-L342)

## Baseline fork model comparison

| Area | `goquarkchain` | `geth 1.17.3` | Replay implication |
| --- | --- | --- | --- |
| Default EVM fork baseline | `DefaultConstantinople` sets Homestead, EIP-150, EIP-155, EIP-158, DAO, Byzantium, and Constantinople at block zero. | Fork-specific jump tables exist from Frontier through Amsterdam-era rules. | A replay engine must not blindly select the newest geth table for old goquarkchain blocks. |
| Jump table coverage | Frontier, Homestead, Byzantium, Constantinople. | Frontier, Homestead, Tangerine Whistle, Spurious Dragon, Byzantium, Constantinople, Istanbul, Berlin, London, Merge, Shanghai, Cancun, Verkle/UBT, Prague, Osaka, Amsterdam. | Later geth tables introduce both new opcodes and repriced existing opcodes. |
| Merge handling | No `PREVRANDAO` alias or Merge jump table. `0x44` remains `DIFFICULTY`. | Merge table remaps `0x44` to `PREVRANDAO`/randomness when Merge rules are active. | Contracts reading `0x44` can see different values. |
| State access model | No EIP-2929 cold/warm access-list gas model in the inspected EVM. | Berlin rules apply cold/warm accounting for SLOAD, account access, call-family opcodes, and selfdestruct. | Gasleft and out-of-gas behavior can diverge even when opcode semantics look similar. |

## Opcode set differences

The following opcodes are present in `geth 1.17.3` source but absent from the inspected `goquarkchain` opcode set. Some are activated by specific fork tables; others are defined in the opcode namespace and only become executable when the corresponding feature table enables them.

| Opcode | Name | Introduced/associated rule in `geth 1.17.3` | `goquarkchain` behavior | Replay risk |
| --- | --- | --- | --- | --- |
| `0x1e` | `CLZ` | Osaka/EIP-7939 | Undefined opcode | Bytecode using it is invalid in goquarkchain but executable in geth when enabled. |
| `0x46` | `CHAINID` | Istanbul/EIP-1344 | Undefined opcode | Environment value becomes observable in geth. |
| `0x47` | `SELFBALANCE` | Istanbul/EIP-1884 | Undefined opcode | Contract self-balance becomes directly readable in geth. |
| `0x48` | `BASEFEE` | London/EIP-3198 | Undefined opcode | Base fee becomes observable in geth. |
| `0x49` | `BLOBHASH` | Cancun/EIP-4844 | Undefined opcode | Blob transaction context becomes observable in geth. |
| `0x4a` | `BLOBBASEFEE` | Cancun/EIP-7516 | Undefined opcode | Blob base fee becomes observable in geth. |
| `0x4b` | `SLOTNUM` | Amsterdam/EIP-7843 | Undefined opcode | Slot number becomes observable in geth. |
| `0x5c` | `TLOAD` | Cancun/EIP-1153 | Undefined opcode | Transient storage reads become executable in geth. |
| `0x5d` | `TSTORE` | Cancun/EIP-1153 | Undefined opcode | Transient storage writes become executable in geth. |
| `0x5e` | `MCOPY` | Cancun/EIP-5656 | Undefined opcode | Memory copy behavior becomes executable in geth. |
| `0x5f` | `PUSH0` | Shanghai/EIP-3855 | Undefined opcode | Common compiler output using `PUSH0` fails in goquarkchain but succeeds in geth. |
| `0xd0`-`0xd3` | `DATALOAD`, `DATALOADN`, `DATASIZE`, `DATACOPY` | EOF-related namespace | Undefined opcode | Must remain disabled for goquarkchain historical replay. |
| `0xe0`-`0xe8` | `RJUMP`, `RJUMPI`, `RJUMPV`, `CALLF`, `RETF`, `JUMPF`, `DUPN`, `SWAPN`, `EXCHANGE` | EOF/Amsterdam-era namespace; `DUPN`, `SWAPN`, `EXCHANGE` are enabled by EIP-8024 | Undefined opcode | Control-flow and stack behavior can diverge if enabled. |
| `0xec`, `0xee` | `EOFCREATE`, `RETURNCONTRACT` | EOF-related namespace | Undefined opcode | Contract creation semantics must not leak into historical replay. |
| `0xf7`-`0xfb` | `RETURNDATALOAD`, `EXTCALL`, `EXTDELEGATECALL`, `EXTSTATICCALL` | New return-data/call-family namespace | Undefined opcode | Call semantics can diverge if enabled. |
| `0xfe` | `INVALID` | Explicit constant in geth | No explicit named constant in the inspected opcode list | Usually equivalent invalid behavior, but traces/names can differ. |

`0x20` is a naming difference rather than a semantic difference: `goquarkchain` names it `SHA3`; `geth 1.17.3` names it `KECCAK256`.

## Same-opcode semantic differences

| Opcode | `goquarkchain` behavior | `geth 1.17.3` behavior | Replay risk |
| --- | --- | --- | --- |
| `0x44 DIFFICULTY` / `PREVRANDAO` | Always pushes the block difficulty from `interpreter.evm.Difficulty`: [instructions.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/core/vm/instructions.go#L591-L593). | Pre-Merge `opDifficulty` pushes difficulty; Merge rules replace the jump table entry with `opRandom`, which pushes `prevRandao`/randomness: [instructions.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/instructions.go#L467-L474). | High. Any contract using `0x44` can compute different values and write different state. |
| `SELFDESTRUCT` | Transfers balance to the beneficiary and marks the contract as suicided: [instructions.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/core/vm/instructions.go#L889-L894). | Pre-Cancun `opSelfdestruct` deletes the account; Cancun/EIP-6780 `opSelfdestruct6780` deletes only newly created contracts in the same transaction and otherwise mostly transfers balance: [instructions.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/instructions.go#L878-L930). | High. Account deletion, code/storage persistence, balance movement, and refunds can diverge. |
| `CREATE` / `CREATE2` | Constantinople-era behavior without EIP-3860 initcode metering or size limit. | Shanghai/EIP-3860 meters initcode by word and checks maximum initcode size: [gas_table.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/gas_table.go#L315-L350). | Medium to high. Large initcode or tight gas can succeed in one engine and fail in the other. |
| `CALL`, `CALLCODE`, `DELEGATECALL`, `STATICCALL` | EIP-150/EIP-158-era gas model, plus QuarkChain-specific token/balance behavior in its EVM context. | Berlin/EIP-2929 adds cold/warm account access costs; Prague/EIP-7702 changes call gas handling for delegation designators: [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L159-L203), [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L267-L292). | High. Gas forwarding and gasleft-sensitive code can diverge. |
| `SSTORE` | The inspected code forces legacy metering through `if true`, using current-state-based 20,000/5,000 gas and 15,000 refund paths: [gas_table.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/core/vm/gas_table.go#L118-L141). | geth 1.17.3 contains EIP-2200, EIP-2929, and EIP-3529 variants, selected by fork rules: [gas_table.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/gas_table.go#L185-L227), [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L208-L224). | Very high. Storage writes, refunds, and out-of-gas behavior are core state-root inputs. |

## Gas metering and refund differences

| Area | `goquarkchain` | `geth 1.17.3` | Compatibility note |
| --- | --- | --- | --- |
| `SLOAD` | Uses the older gas table selected by the old chain rules. | Istanbul reprices SLOAD, and Berlin changes it to cold/warm storage access charging: [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L96-L112). | Historical replay should use the old fixed-cost model unless the chain explicitly migrated. |
| Account access opcodes | `BALANCE`, `EXTCODESIZE`, `EXTCODEHASH`, and `EXTCODECOPY` use old fixed-style gas table behavior. | Istanbul reprices `BALANCE`, `EXTCODEHASH`, and `SLOAD`; Berlin adds cold/warm access checks for account reads and extcode operations: [eips.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/eips.go#L73-L90), [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L114-L157). | Cold/warm access accounting can change whether later opcodes have enough gas. |
| Call-family gas | Old call gas model using the selected `GasTable`. | EIP-2929 charges cold account access; EIP-7702 adds delegation-related call gas handling. | Replay should not enable EIP-2929/EIP-7702 for historical goquarkchain blocks unless intentionally forked. |
| `SSTORE` refund | Legacy clear refund is 15,000 in the inspected path. | EIP-3529 lowers clear refund behavior and changes refund cap assumptions. | Refund differences affect effective gas used and can affect block validity/accounting. |
| `SELFDESTRUCT` refund | Adds suicide refund when the account has not suicided before: [gas_table.go](https://github.com/QuarkChain/goquarkchain/blob/8534eaf6f0e64374c81a939901058d84ec285661/core/vm/gas_table.go#L460-L483). | EIP-3529 removes selfdestruct refunds in the London table path: [operations_acl.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/operations_acl.go#L199-L206). | Must preserve old refund behavior for old blocks. |
| Initcode metering | No EIP-3860 metering in the inspected old EVM. | Shanghai adds initcode size checks and per-word gas: [gas_table.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/gas_table.go#L315-L350). | Historical contract creation must not be rejected by newer initcode limits. |
| Memory copy | No `MCOPY`. | Cancun adds `MCOPY` with memory expansion plus copy-word gas: [eips.go](https://github.com/ethereum/go-ethereum/blob/v1.17.3/core/vm/eips.go#L250-L274). | Treat `0x5e` as invalid for old blocks. |

## Recommended compatibility strategy

For historical `goquarkchain` replay, do not execute old blocks with the latest `geth 1.17.3` jump table. Instead, introduce an explicit compatibility ruleset that preserves the old execution surface:

1. Select a `goquarkchain`/Constantinople-compatible jump table for historical blocks.
2. Keep `0x44` mapped to block difficulty, not `PREVRANDAO`.
3. Treat post-Constantinople opcodes such as `CHAINID`, `BASEFEE`, `PUSH0`, `TLOAD`, `TSTORE`, `MCOPY`, blob opcodes, EOF opcodes, and new call-family opcodes as invalid unless a QuarkChain-specific migration height enables them.
4. Preserve old `SLOAD`, `SSTORE`, call-family, and `SELFDESTRUCT` gas/refund behavior for historical blocks.
5. Add replay tests around contracts that exercise `0x44`, `SSTORE`, `SELFDESTRUCT`, `PUSH0`, `BASEFEE`, `TLOAD/TSTORE`, `MCOPY`, and cold/warm access-sensitive calls.

The safest migration model is height-gated execution: old blocks use a pinned `goquarkchain` compatibility EVM, and only blocks after an explicit protocol upgrade height use newer `geth 1.17.3` semantics.
