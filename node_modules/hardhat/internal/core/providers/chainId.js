"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ChainIdValidatorProvider = exports.ProviderWrapperWithChainId = void 0;
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const base_types_1 = require("../jsonrpc/types/base-types");
const wrapper_1 = require("./wrapper");
class ProviderWrapperWithChainId extends wrapper_1.ProviderWrapper {
    async _getChainId() {
        if (this._chainId === undefined) {
            try {
                this._chainId = await this._getChainIdFromEthChainId();
            }
            catch {
                // If eth_chainId fails we default to net_version
                this._chainId = await this._getChainIdFromEthNetVersion();
            }
        }
        return this._chainId;
    }
    async _getChainIdFromEthChainId() {
        const id = (await this._wrappedProvider.request({
            method: "eth_chainId",
        }));
        return (0, base_types_1.rpcQuantityToNumber)(id);
    }
    async _getChainIdFromEthNetVersion() {
        const id = (await this._wrappedProvider.request({
            method: "net_version",
        }));
        // There's a node returning this as decimal instead of QUANTITY.
        // TODO: Document here which node does that
        return id.startsWith("0x") ? (0, base_types_1.rpcQuantityToNumber)(id) : parseInt(id, 10);
    }
}
exports.ProviderWrapperWithChainId = ProviderWrapperWithChainId;
class ChainIdValidatorProvider extends ProviderWrapperWithChainId {
    constructor(provider, _expectedChainId) {
        super(provider);
        this._expectedChainId = _expectedChainId;
        this._alreadyValidated = false;
    }
    async request(args) {
        if (!this._alreadyValidated) {
            const actualChainId = await this._getChainId();
            if (actualChainId !== this._expectedChainId) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.INVALID_GLOBAL_CHAIN_ID, {
                    configChainId: this._expectedChainId,
                    connectionChainId: actualChainId,
                });
            }
            this._alreadyValidated = true;
        }
        return this._wrappedProvider.request(args);
    }
}
exports.ChainIdValidatorProvider = ChainIdValidatorProvider;
//# sourceMappingURL=chainId.js.map