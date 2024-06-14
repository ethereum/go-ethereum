"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.keccak512 = exports.keccak384 = exports.keccak256 = exports.keccak224 = void 0;
const sha3_1 = require("@noble/hashes/sha3");
const utils_1 = require("./utils");
exports.keccak224 = (0, utils_1.wrapHash)(sha3_1.keccak_224);
exports.keccak256 = (() => {
    const k = (0, utils_1.wrapHash)(sha3_1.keccak_256);
    k.create = sha3_1.keccak_256.create;
    return k;
})();
exports.keccak384 = (0, utils_1.wrapHash)(sha3_1.keccak_384);
exports.keccak512 = (0, utils_1.wrapHash)(sha3_1.keccak_512);
