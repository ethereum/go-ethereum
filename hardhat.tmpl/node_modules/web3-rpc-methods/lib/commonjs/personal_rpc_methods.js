"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ecRecover = exports.sign = exports.signTransaction = exports.sendTransaction = exports.importRawKey = exports.lockAccount = exports.unlockAccount = exports.newAccount = exports.getAccounts = void 0;
const getAccounts = (requestManager) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_listAccounts',
        params: [],
    });
});
exports.getAccounts = getAccounts;
const newAccount = (requestManager, password) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_newAccount',
        params: [password],
    });
});
exports.newAccount = newAccount;
const unlockAccount = (requestManager, address, password, unlockDuration) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_unlockAccount',
        params: [address, password, unlockDuration],
    });
});
exports.unlockAccount = unlockAccount;
const lockAccount = (requestManager, address) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_lockAccount',
        params: [address],
    });
});
exports.lockAccount = lockAccount;
const importRawKey = (requestManager, keyData, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_importRawKey',
        params: [keyData, passphrase],
    });
});
exports.importRawKey = importRawKey;
const sendTransaction = (requestManager, tx, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_sendTransaction',
        params: [tx, passphrase],
    });
});
exports.sendTransaction = sendTransaction;
const signTransaction = (requestManager, tx, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_signTransaction',
        params: [tx, passphrase],
    });
});
exports.signTransaction = signTransaction;
const sign = (requestManager, data, address, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_sign',
        params: [data, address, passphrase],
    });
});
exports.sign = sign;
const ecRecover = (requestManager, signedData, signature) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_ecRecover',
        params: [signedData, signature],
    });
});
exports.ecRecover = ecRecover;
//# sourceMappingURL=personal_rpc_methods.js.map