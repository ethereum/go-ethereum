"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pbkdf2 = void 0;
var bytes_1 = require("@ethersproject/bytes");
var sha2_1 = require("@ethersproject/sha2");
function pbkdf2(password, salt, iterations, keylen, hashAlgorithm) {
    password = (0, bytes_1.arrayify)(password);
    salt = (0, bytes_1.arrayify)(salt);
    var hLen;
    var l = 1;
    var DK = new Uint8Array(keylen);
    var block1 = new Uint8Array(salt.length + 4);
    block1.set(salt);
    //salt.copy(block1, 0, 0, salt.length)
    var r;
    var T;
    for (var i = 1; i <= l; i++) {
        //block1.writeUInt32BE(i, salt.length)
        block1[salt.length] = (i >> 24) & 0xff;
        block1[salt.length + 1] = (i >> 16) & 0xff;
        block1[salt.length + 2] = (i >> 8) & 0xff;
        block1[salt.length + 3] = i & 0xff;
        //let U = createHmac(password).update(block1).digest();
        var U = (0, bytes_1.arrayify)((0, sha2_1.computeHmac)(hashAlgorithm, password, block1));
        if (!hLen) {
            hLen = U.length;
            T = new Uint8Array(hLen);
            l = Math.ceil(keylen / hLen);
            r = keylen - (l - 1) * hLen;
        }
        //U.copy(T, 0, 0, hLen)
        T.set(U);
        for (var j = 1; j < iterations; j++) {
            //U = createHmac(password).update(U).digest();
            U = (0, bytes_1.arrayify)((0, sha2_1.computeHmac)(hashAlgorithm, password, U));
            for (var k = 0; k < hLen; k++)
                T[k] ^= U[k];
        }
        var destPos = (i - 1) * hLen;
        var len = (i === l ? r : hLen);
        //T.copy(DK, destPos, 0, len)
        DK.set((0, bytes_1.arrayify)(T).slice(0, len), destPos);
    }
    return (0, bytes_1.hexlify)(DK);
}
exports.pbkdf2 = pbkdf2;
//# sourceMappingURL=browser-pbkdf2.js.map