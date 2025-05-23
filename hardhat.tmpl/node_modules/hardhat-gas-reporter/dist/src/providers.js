"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EGRAsyncApiProvider = exports.EGRDataCollectionProvider = void 0;
const wrapper_1 = require("hardhat/internal/core/providers/wrapper");
/**
 * Wrapped provider which collects tx data
 */
class EGRDataCollectionProvider extends wrapper_1.ProviderWrapper {
    constructor(provider, mochaConfig) {
        super(provider);
        this.mochaConfig = mochaConfig;
    }
    async request(args) {
        // Truffle
        if (args.method === "eth_getTransactionReceipt") {
            const receipt = await this._wrappedProvider.request(args);
            if ((receipt === null || receipt === void 0 ? void 0 : receipt.status) && (receipt === null || receipt === void 0 ? void 0 : receipt.transactionHash)) {
                const tx = await this._wrappedProvider.request({
                    method: "eth_getTransactionByHash",
                    params: [receipt.transactionHash]
                });
                await this.mochaConfig.attachments.recordTransaction(receipt, tx);
            }
            return receipt;
            // Ethers: will get run twice for deployments (e.g both receipt and txhash are fetched)
        }
        else if (args.method === 'eth_getTransactionByHash') {
            const receipt = await this._wrappedProvider.request({
                method: "eth_getTransactionReceipt",
                params: args.params
            });
            const tx = await this._wrappedProvider.request(args);
            if (receipt === null || receipt === void 0 ? void 0 : receipt.status) {
                await this.mochaConfig.attachments.recordTransaction(receipt, tx);
            }
            return tx;
            // Waffle: This is necessary when using Waffle wallets. eth_sendTransaction fetches the
            // transactionHash as part of its flow, eth_sendRawTransaction *does not*.
        }
        else if (args.method === 'eth_sendRawTransaction') {
            const txHash = await this._wrappedProvider.request(args);
            if (typeof txHash === 'string') {
                const tx = await this._wrappedProvider.request({
                    method: "eth_getTransactionByHash",
                    params: [txHash]
                });
                const receipt = await this._wrappedProvider.request({
                    method: "eth_getTransactionReceipt",
                    params: [txHash]
                });
                if (receipt === null || receipt === void 0 ? void 0 : receipt.status) {
                    await this.mochaConfig.attachments.recordTransaction(receipt, tx);
                }
            }
            return txHash;
        }
        return this._wrappedProvider.request(args);
    }
}
exports.EGRDataCollectionProvider = EGRDataCollectionProvider;
/**
 * A set of async RPC calls which substitute the sync calls made by the core reporter
 * These allow us to use HardhatEVM or another in-process provider.
 */
class EGRAsyncApiProvider {
    constructor(provider) {
        this.provider = provider;
    }
    async getNetworkId() {
        return this.provider.send("net_version", []);
    }
    async getCode(address) {
        return this.provider.send("eth_getCode", [address, "latest"]);
    }
    async getLatestBlock() {
        return this.provider.send("eth_getBlockByNumber", ["latest", false]);
    }
    async getBlockByNumber(num) {
        const hexNumber = `0x${num.toString(16)}`;
        return this.provider.send("eth_getBlockByNumber", [hexNumber, true]);
    }
    async blockNumber() {
        const block = await this.getLatestBlock();
        return parseInt(block.number, 16);
    }
    async getTransactionByHash(tx) {
        return this.provider.send("eth_getTransactionByHash", [tx]);
    }
    async call(payload, blockNumber) {
        return this.provider.send("eth_call", [payload, blockNumber]);
    }
}
exports.EGRAsyncApiProvider = EGRAsyncApiProvider;
//# sourceMappingURL=providers.js.map