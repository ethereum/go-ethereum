"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseGethGenesisState = void 0;
const bytes_js_1 = require("./bytes.js");
const internal_js_1 = require("./internal.js");
/**
 * Parses the geth genesis state into Blockchain {@link GenesisState}
 * @param json representing the `alloc` key in a Geth genesis file
 */
function parseGethGenesisState(json) {
    const state = {};
    for (const address of Object.keys(json.alloc)) {
        let { balance, code, storage, nonce } = json.alloc[address];
        // create a map with lowercase for easy lookups
        const prefixedAddress = (0, bytes_js_1.addHexPrefix)(address.toLowerCase());
        balance = (0, internal_js_1.isHexString)(balance) ? balance : (0, bytes_js_1.bigIntToHex)(BigInt(balance));
        code = code !== undefined ? (0, bytes_js_1.addHexPrefix)(code) : undefined;
        storage = storage !== undefined ? Object.entries(storage) : undefined;
        nonce = nonce !== undefined ? (0, bytes_js_1.addHexPrefix)(nonce) : undefined;
        state[prefixedAddress] = [balance, code, storage, nonce];
    }
    return state;
}
exports.parseGethGenesisState = parseGethGenesisState;
//# sourceMappingURL=genesis.js.map