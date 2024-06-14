"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deserializeReplacer = void 0;
/**
 * When JSON.parsing journal messages deserialize, this defines the replacer.
 */
function deserializeReplacer(_key, value) {
    if (typeof value === "string" && /^\d+n$/.test(value)) {
        return BigInt(value.slice(0, -1));
    }
    if (_isSerializedBigInt(value)) {
        return BigInt(value.value);
    }
    return value;
}
exports.deserializeReplacer = deserializeReplacer;
function _isSerializedBigInt(arg) {
    return (arg !== null &&
        typeof arg === "object" &&
        "_kind" in arg &&
        arg._kind === "bigint");
}
//# sourceMappingURL=deserialize-replacer.js.map