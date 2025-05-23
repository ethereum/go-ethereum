var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
export function getId(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'net_version',
            params: [],
        });
    });
}
export function getPeerCount(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'net_peerCount',
            params: [],
        });
    });
}
export function isListening(requestManager) {
    return __awaiter(this, void 0, void 0, function* () {
        return requestManager.send({
            method: 'net_listening',
            params: [],
        });
    });
}
//# sourceMappingURL=net_rpc_methods.js.map