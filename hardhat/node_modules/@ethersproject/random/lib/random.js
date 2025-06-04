"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.randomBytes = void 0;
var crypto_1 = require("crypto");
var bytes_1 = require("@ethersproject/bytes");
function randomBytes(length) {
    return (0, bytes_1.arrayify)((0, crypto_1.randomBytes)(length));
}
exports.randomBytes = randomBytes;
//# sourceMappingURL=random.js.map