"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var ripemd160_1 = require("../ripemd160");
var sha256_1 = require("../sha256");
exports.createHmac = require("create-hmac");
exports.randomBytes = require("randombytes");
var Hash = /** @class */ (function () {
    function Hash(hashFunction) {
        this.hashFunction = hashFunction;
        this.buffers = [];
    }
    Hash.prototype.update = function (buffer) {
        if (!Buffer.isBuffer(buffer)) {
            throw new Error("hdkey-crypto shim is outdated");
        }
        this.buffers.push(buffer);
        return this;
    };
    Hash.prototype.digest = function (param) {
        if (param) {
            throw new Error("hdkey-crypto shim is outdated");
        }
        return this.hashFunction(Buffer.concat(this.buffers));
    };
    return Hash;
}());
// We don't use create-hash here, as it doesn't work well with Rollup
exports.createHash = function (name) {
    if (name === "ripemd160") {
        return new Hash(ripemd160_1.ripemd160);
    }
    if (name === "sha256") {
        return new Hash(sha256_1.sha256);
    }
    throw new Error("hdkey-crypto shim is outdated");
};
//# sourceMappingURL=hdkey-crypto.js.map