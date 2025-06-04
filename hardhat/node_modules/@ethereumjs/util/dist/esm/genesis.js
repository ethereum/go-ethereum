import { addHexPrefix, bigIntToHex } from './bytes.js';
import { isHexString } from './internal.js';
/**
 * Parses the geth genesis state into Blockchain {@link GenesisState}
 * @param json representing the `alloc` key in a Geth genesis file
 */
export function parseGethGenesisState(json) {
    const state = {};
    for (const address of Object.keys(json.alloc)) {
        let { balance, code, storage, nonce } = json.alloc[address];
        // create a map with lowercase for easy lookups
        const prefixedAddress = addHexPrefix(address.toLowerCase());
        balance = isHexString(balance) ? balance : bigIntToHex(BigInt(balance));
        code = code !== undefined ? addHexPrefix(code) : undefined;
        storage = storage !== undefined ? Object.entries(storage) : undefined;
        nonce = nonce !== undefined ? addHexPrefix(nonce) : undefined;
        state[prefixedAddress] = [balance, code, storage, nonce];
    }
    return state;
}
//# sourceMappingURL=genesis.js.map