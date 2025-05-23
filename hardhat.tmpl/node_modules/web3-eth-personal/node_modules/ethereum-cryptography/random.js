"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getRandomBytesSync = getRandomBytesSync;
exports.getRandomBytes = getRandomBytes;
const utils_1 = require("@noble/hashes/utils");
function getRandomBytesSync(bytes) {
    return (0, utils_1.randomBytes)(bytes);
}
async function getRandomBytes(bytes) {
    return (0, utils_1.randomBytes)(bytes);
}
