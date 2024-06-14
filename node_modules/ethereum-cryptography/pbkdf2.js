"use strict";
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (Object.hasOwnProperty.call(mod, k)) result[k] = mod[k];
    result["default"] = mod;
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
var pbkdf2Js = __importStar(require("pbkdf2"));
function pbkdf2(password, salt, iterations, keylen, digest) {
    return new Promise(function (resolve, reject) {
        pbkdf2Js.pbkdf2(password, salt, iterations, keylen, digest, function (err, result) {
            if (err) {
                reject(err);
                return;
            }
            resolve(result);
        });
    });
}
exports.pbkdf2 = pbkdf2;
function pbkdf2Sync(password, salt, iterations, keylen, digest) {
    return pbkdf2Js.pbkdf2Sync(password, salt, iterations, keylen, digest);
}
exports.pbkdf2Sync = pbkdf2Sync;
//# sourceMappingURL=pbkdf2.js.map