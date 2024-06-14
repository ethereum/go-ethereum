"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decryptJsonWalletSync = exports.decryptJsonWallet = exports.getJsonWalletAddress = exports.isKeystoreWallet = exports.isCrowdsaleWallet = exports.encryptKeystore = exports.decryptKeystoreSync = exports.decryptKeystore = exports.decryptCrowdsale = void 0;
var crowdsale_1 = require("./crowdsale");
Object.defineProperty(exports, "decryptCrowdsale", { enumerable: true, get: function () { return crowdsale_1.decrypt; } });
var inspect_1 = require("./inspect");
Object.defineProperty(exports, "getJsonWalletAddress", { enumerable: true, get: function () { return inspect_1.getJsonWalletAddress; } });
Object.defineProperty(exports, "isCrowdsaleWallet", { enumerable: true, get: function () { return inspect_1.isCrowdsaleWallet; } });
Object.defineProperty(exports, "isKeystoreWallet", { enumerable: true, get: function () { return inspect_1.isKeystoreWallet; } });
var keystore_1 = require("./keystore");
Object.defineProperty(exports, "decryptKeystore", { enumerable: true, get: function () { return keystore_1.decrypt; } });
Object.defineProperty(exports, "decryptKeystoreSync", { enumerable: true, get: function () { return keystore_1.decryptSync; } });
Object.defineProperty(exports, "encryptKeystore", { enumerable: true, get: function () { return keystore_1.encrypt; } });
function decryptJsonWallet(json, password, progressCallback) {
    if ((0, inspect_1.isCrowdsaleWallet)(json)) {
        if (progressCallback) {
            progressCallback(0);
        }
        var account = (0, crowdsale_1.decrypt)(json, password);
        if (progressCallback) {
            progressCallback(1);
        }
        return Promise.resolve(account);
    }
    if ((0, inspect_1.isKeystoreWallet)(json)) {
        return (0, keystore_1.decrypt)(json, password, progressCallback);
    }
    return Promise.reject(new Error("invalid JSON wallet"));
}
exports.decryptJsonWallet = decryptJsonWallet;
function decryptJsonWalletSync(json, password) {
    if ((0, inspect_1.isCrowdsaleWallet)(json)) {
        return (0, crowdsale_1.decrypt)(json, password);
    }
    if ((0, inspect_1.isKeystoreWallet)(json)) {
        return (0, keystore_1.decryptSync)(json, password);
    }
    throw new Error("invalid JSON wallet");
}
exports.decryptJsonWalletSync = decryptJsonWalletSync;
//# sourceMappingURL=index.js.map