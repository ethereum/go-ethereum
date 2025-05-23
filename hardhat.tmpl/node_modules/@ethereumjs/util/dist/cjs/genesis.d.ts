import type { PrefixedHexString } from './types.js';
export declare type StoragePair = [key: PrefixedHexString, value: PrefixedHexString];
export declare type AccountState = [
    balance: PrefixedHexString,
    code: PrefixedHexString,
    storage: Array<StoragePair>,
    nonce: PrefixedHexString
];
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
    [key: string]: PrefixedHexString | AccountState;
}
/**
 * Parses the geth genesis state into Blockchain {@link GenesisState}
 * @param json representing the `alloc` key in a Geth genesis file
 */
export declare function parseGethGenesisState(json: any): GenesisState;
//# sourceMappingURL=genesis.d.ts.map