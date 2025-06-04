"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha512_256 = exports.sha512_224 = exports.sha512 = exports.sha384 = exports.sha256 = exports.sha224 = void 0;
/**
 * SHA2 hash function. A.k.a. sha256, sha512, sha512_256, etc.
 * @module
 */
// Usually you either use sha256, or sha512. We re-export them as sha2 for naming consistency.
var sha256_ts_1 = require("./sha256.js");
Object.defineProperty(exports, "sha224", { enumerable: true, get: function () { return sha256_ts_1.sha224; } });
Object.defineProperty(exports, "sha256", { enumerable: true, get: function () { return sha256_ts_1.sha256; } });
var sha512_ts_1 = require("./sha512.js");
Object.defineProperty(exports, "sha384", { enumerable: true, get: function () { return sha512_ts_1.sha384; } });
Object.defineProperty(exports, "sha512", { enumerable: true, get: function () { return sha512_ts_1.sha512; } });
Object.defineProperty(exports, "sha512_224", { enumerable: true, get: function () { return sha512_ts_1.sha512_224; } });
Object.defineProperty(exports, "sha512_256", { enumerable: true, get: function () { return sha512_ts_1.sha512_256; } });
//# sourceMappingURL=sha2.js.map