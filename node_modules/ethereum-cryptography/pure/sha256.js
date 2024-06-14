"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Sha256Hash = require("hash.js/lib/hash/sha/256");
var hash_utils_1 = require("../hash-utils");
exports.sha256 = hash_utils_1.createHashFunction(function () { return new Sha256Hash(); });
//# sourceMappingURL=sha256.js.map