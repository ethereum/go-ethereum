import { addHexPrefix, bigIntToHex } from './bytes.js'
import { isHexPrefixed } from './internal.js'

import type { PrefixedHexString } from './types.js'

export type StoragePair = [key: PrefixedHexString, value: PrefixedHexString]

export type AccountState = [
  balance: PrefixedHexString,
  code: PrefixedHexString,
  storage: Array<StoragePair>,
  nonce: PrefixedHexString
]

/**
 * If you are using a custom chain {@link Common}, pass the genesis state.
 *
 * Pattern 1 (with genesis state see {@link GenesisState} for format):
 *
 * ```javascript
 * {
 *   '0x0...01': '0x100', // For EoA
 * }
 * ```
 *
 * Pattern 2 (with complex genesis state, containing contract accounts and storage).
 * Note that in {@link AccountState} there are two
 * accepted types. This allows to easily insert accounts in the genesis state:
 *
 * A complex genesis state with Contract and EoA states would have the following format:
 *
 * ```javascript
 * {
 *   '0x0...01': '0x100', // For EoA
 *   '0x0...02': ['0x1', '0xRUNTIME_BYTECODE', [[storageKey1, storageValue1], [storageKey2, storageValue2]]] // For contracts
 * }
 * ```
 */
export interface GenesisState {
  [key: PrefixedHexString]: PrefixedHexString | AccountState
}

/**
 * Parses the geth genesis state into Blockchain {@link GenesisState}
 * @param json representing the `alloc` key in a Geth genesis file
 */
export function parseGethGenesisState(json: any) {
  const state: GenesisState = {}
  for (let address of Object.keys(json.alloc)) {
    let { balance, code, storage, nonce } = json.alloc[address]
    // create a map with lowercase for easy lookups
    address = addHexPrefix(address.toLowerCase())
    balance = isHexPrefixed(balance) ? balance : bigIntToHex(BigInt(balance))
    code = code !== undefined ? addHexPrefix(code) : undefined
    storage = storage !== undefined ? Object.entries(storage) : undefined
    nonce = nonce !== undefined ? addHexPrefix(nonce) : undefined
    state[address] = [balance, code, storage, nonce]
  }
  return state
}
