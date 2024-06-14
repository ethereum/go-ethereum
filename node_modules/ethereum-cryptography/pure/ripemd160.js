"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Ripemd160 = require("hash.js/lib/hash/ripemd").ripemd160;
var hash_utils_1 = require("../hash-utils");
exports.ripemd160 = hash_utils_1.createHashFunction(function () { return new Ripemd160(); });
//# sourceMappingURL=ripemd160.js.map