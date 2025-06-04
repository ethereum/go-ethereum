"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var blake2bJs = require("blakejs");
function blake2b(input, outputLength) {
    if (outputLength === void 0) { outputLength = 64; }
    if (outputLength <= 0 || outputLength > 64) {
        throw Error("Invalid outputLength");
    }
    return Buffer.from(blake2bJs.blake2b(input, undefined, outputLength));
}
exports.blake2b = blake2b;
//# sourceMappingURL=blake2b.js.map