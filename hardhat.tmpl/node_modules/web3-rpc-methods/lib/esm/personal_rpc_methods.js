var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
export const getAccounts = (requestManager) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_listAccounts',
        params: [],
    });
});
export const newAccount = (requestManager, password) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_newAccount',
        params: [password],
    });
});
export const unlockAccount = (requestManager, address, password, unlockDuration) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_unlockAccount',
        params: [address, password, unlockDuration],
    });
});
export const lockAccount = (requestManager, address) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_lockAccount',
        params: [address],
    });
});
export const importRawKey = (requestManager, keyData, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_importRawKey',
        params: [keyData, passphrase],
    });
});
export const sendTransaction = (requestManager, tx, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_sendTransaction',
        params: [tx, passphrase],
    });
});
export const signTransaction = (requestManager, tx, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_signTransaction',
        params: [tx, passphrase],
    });
});
export const sign = (requestManager, data, address, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_sign',
        params: [data, address, passphrase],
    });
});
export const ecRecover = (requestManager, signedData, signature) => __awaiter(void 0, void 0, void 0, function* () {
    return requestManager.send({
        method: 'personal_ecRecover',
        params: [signedData, signature],
    });
});
//# sourceMappingURL=personal_rpc_methods.js.map