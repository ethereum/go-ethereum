"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.blake2b = void 0;
const blake2b_1 = require("@noble/hashes/blake2b");
const utils_1 = require("./utils");
const blake2b = (msg, outputLength = 64) => {
    (0, utils_1.assertBytes)(msg);
    if (outputLength <= 0 || outputLength > 64) {
        throw Error("Invalid outputLength");
    }
    return (0, blake2b_1.blake2b)(msg, { dkLen: outputLength });
};
exports.blake2b = blake2b;
