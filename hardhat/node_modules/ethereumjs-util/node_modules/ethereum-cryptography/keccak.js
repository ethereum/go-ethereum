"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var hash_utils_1 = require("./hash-utils");
var createKeccakHash = require("keccak");
exports.keccak224 = hash_utils_1.createHashFunction(function () {
    return createKeccakHash("keccak224");
});
exports.keccak256 = hash_utils_1.createHashFunction(function () {
    return createKeccakHash("keccak256");
});
exports.keccak384 = hash_utils_1.createHashFunction(function () {
    return createKeccakHash("keccak384");
});
exports.keccak512 = hash_utils_1.createHashFunction(function () {
    return createKeccakHash("keccak512");
});
//# sourceMappingURL=keccak.js.map