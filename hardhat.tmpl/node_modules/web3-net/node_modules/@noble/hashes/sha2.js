"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha384 = exports.sha512_256 = exports.sha512_224 = exports.sha512 = exports.sha224 = exports.sha256 = void 0;
// Usually you either use sha256, or sha512. We re-export them as sha2 for naming consistency.
var sha256_js_1 = require("./sha256.js");
Object.defineProperty(exports, "sha256", { enumerable: true, get: function () { return sha256_js_1.sha256; } });
Object.defineProperty(exports, "sha224", { enumerable: true, get: function () { return sha256_js_1.sha224; } });
var sha512_js_1 = require("./sha512.js");
Object.defineProperty(exports, "sha512", { enumerable: true, get: function () { return sha512_js_1.sha512; } });
Object.defineProperty(exports, "sha512_224", { enumerable: true, get: function () { return sha512_js_1.sha512_224; } });
Object.defineProperty(exports, "sha512_256", { enumerable: true, get: function () { return sha512_js_1.sha512_256; } });
Object.defineProperty(exports, "sha384", { enumerable: true, get: function () { return sha512_js_1.sha384; } });
//# sourceMappingURL=sha2.js.map