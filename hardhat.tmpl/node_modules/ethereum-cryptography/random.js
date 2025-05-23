"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getRandomBytes = exports.getRandomBytesSync = void 0;
const utils_1 = require("@noble/hashes/utils");
function getRandomBytesSync(bytes) {
    return (0, utils_1.randomBytes)(bytes);
}
exports.getRandomBytesSync = getRandomBytesSync;
async function getRandomBytes(bytes) {
    return (0, utils_1.randomBytes)(bytes);
}
exports.getRandomBytes = getRandomBytes;
