"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.bigintReviver = void 0;
const plugins_1 = require("hardhat/plugins");
function bigintReviver(key, value) {
    if (typeof value === "string" && /^\d+n$/.test(value)) {
        return BigInt(value.slice(0, -1));
    }
    if (typeof value === "number" && value > Number.MAX_SAFE_INTEGER) {
        throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", `Parameter "${key}" exceeds maximum safe integer size. Encode the value as a string using bigint notation: \`$\{value\}n\``);
    }
    return value;
}
exports.bigintReviver = bigintReviver;
//# sourceMappingURL=bigintReviver.js.map