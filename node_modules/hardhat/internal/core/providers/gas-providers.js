"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AutomaticGasPriceProvider = exports.AutomaticGasProvider = exports.FixedGasPriceProvider = exports.FixedGasProvider = void 0;
const base_types_1 = require("../jsonrpc/types/base-types");
const wrapper_1 = require("./wrapper");
const DEFAULT_GAS_MULTIPLIER = 1;
class FixedGasProvider extends wrapper_1.ProviderWrapper {
    constructor(provider, _gasLimit) {
        super(provider);
        this._gasLimit = _gasLimit;
    }
    async request(args) {
        if (args.method === "eth_sendTransaction") {
            const params = this._getParams(args);
            // TODO: Should we validate this type?
            const tx = params[0];
            if (tx !== undefined && tx.gas === undefined) {
                tx.gas = (0, base_types_1.numberToRpcQuantity)(this._gasLimit);
            }
        }
        return this._wrappedProvider.request(args);
    }
}
exports.FixedGasProvider = FixedGasProvider;
class FixedGasPriceProvider extends wrapper_1.ProviderWrapper {
    constructor(provider, _gasPrice) {
        super(provider);
        this._gasPrice = _gasPrice;
    }
    async request(args) {
        if (args.method === "eth_sendTransaction") {
            const params = this._getParams(args);
            // TODO: Should we validate this type?
            const tx = params[0];
            // temporary change to ignore EIP-1559
            if (tx !== undefined &&
                tx.gasPrice === undefined &&
                tx.maxFeePerGas === undefined &&
                tx.maxPriorityFeePerGas === undefined) {
                tx.gasPrice = (0, base_types_1.numberToRpcQuantity)(this._gasPrice);
            }
        }
        return this._wrappedProvider.request(args);
    }
}
exports.FixedGasPriceProvider = FixedGasPriceProvider;
class MultipliedGasEstimationProvider extends wrapper_1.ProviderWrapper {
    constructor(provider, _gasMultiplier) {
        super(provider);
        this._gasMultiplier = _gasMultiplier;
    }
    async _getMultipliedGasEstimation(params) {
        try {
            const realEstimation = (await this._wrappedProvider.request({
                method: "eth_estimateGas",
                params,
            }));
            if (this._gasMultiplier === 1) {
                return realEstimation;
            }
            const normalGas = (0, base_types_1.rpcQuantityToNumber)(realEstimation);
            const gasLimit = await this._getBlockGasLimit();
            const multiplied = Math.floor(normalGas * this._gasMultiplier);
            const gas = multiplied > gasLimit ? gasLimit - 1 : multiplied;
            return (0, base_types_1.numberToRpcQuantity)(gas);
        }
        catch (error) {
            if (error instanceof Error) {
                if (error.message.toLowerCase().includes("execution error")) {
                    const blockGasLimit = await this._getBlockGasLimit();
                    return (0, base_types_1.numberToRpcQuantity)(blockGasLimit);
                }
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    }
    async _getBlockGasLimit() {
        if (this._blockGasLimit === undefined) {
            const latestBlock = (await this._wrappedProvider.request({
                method: "eth_getBlockByNumber",
                params: ["latest", false],
            }));
            const fetchedGasLimit = (0, base_types_1.rpcQuantityToNumber)(latestBlock.gasLimit);
            // We store a lower value in case the gas limit varies slightly
            this._blockGasLimit = Math.floor(fetchedGasLimit * 0.95);
        }
        return this._blockGasLimit;
    }
}
class AutomaticGasProvider extends MultipliedGasEstimationProvider {
    constructor(provider, gasMultiplier = DEFAULT_GAS_MULTIPLIER) {
        super(provider, gasMultiplier);
    }
    async request(args) {
        if (args.method === "eth_sendTransaction") {
            const params = this._getParams(args);
            // TODO: Should we validate this type?
            const tx = params[0];
            if (tx !== undefined && tx.gas === undefined) {
                tx.gas = await this._getMultipliedGasEstimation(params);
            }
        }
        return this._wrappedProvider.request(args);
    }
}
exports.AutomaticGasProvider = AutomaticGasProvider;
class AutomaticGasPriceProvider extends wrapper_1.ProviderWrapper {
    async request(args) {
        if (args.method !== "eth_sendTransaction") {
            return this._wrappedProvider.request(args);
        }
        const params = this._getParams(args);
        // TODO: Should we validate this type?
        const tx = params[0];
        if (tx === undefined) {
            return this._wrappedProvider.request(args);
        }
        // We don't need to do anything in these cases
        if (tx.gasPrice !== undefined ||
            (tx.maxFeePerGas !== undefined && tx.maxPriorityFeePerGas !== undefined)) {
            return this._wrappedProvider.request(args);
        }
        let suggestedEip1559Values = await this._suggestEip1559FeePriceValues();
        // eth_feeHistory failed, so we send a legacy one
        if (tx.maxFeePerGas === undefined &&
            tx.maxPriorityFeePerGas === undefined &&
            suggestedEip1559Values === undefined) {
            tx.gasPrice = (0, base_types_1.numberToRpcQuantity)(await this._getGasPrice());
            return this._wrappedProvider.request(args);
        }
        // If eth_feeHistory failed, but the user still wants to send an EIP-1559 tx
        // we use the gasPrice as default values.
        if (suggestedEip1559Values === undefined) {
            const gasPrice = await this._getGasPrice();
            suggestedEip1559Values = {
                maxFeePerGas: gasPrice,
                maxPriorityFeePerGas: gasPrice,
            };
        }
        let maxFeePerGas = tx.maxFeePerGas !== undefined
            ? (0, base_types_1.rpcQuantityToBigInt)(tx.maxFeePerGas)
            : suggestedEip1559Values.maxFeePerGas;
        const maxPriorityFeePerGas = tx.maxPriorityFeePerGas !== undefined
            ? (0, base_types_1.rpcQuantityToBigInt)(tx.maxPriorityFeePerGas)
            : suggestedEip1559Values.maxPriorityFeePerGas;
        if (maxFeePerGas < maxPriorityFeePerGas) {
            maxFeePerGas += maxPriorityFeePerGas;
        }
        tx.maxFeePerGas = (0, base_types_1.numberToRpcQuantity)(maxFeePerGas);
        tx.maxPriorityFeePerGas = (0, base_types_1.numberToRpcQuantity)(maxPriorityFeePerGas);
        return this._wrappedProvider.request(args);
    }
    async _getGasPrice() {
        const response = (await this._wrappedProvider.request({
            method: "eth_gasPrice",
        }));
        return (0, base_types_1.rpcQuantityToBigInt)(response);
    }
    async _suggestEip1559FeePriceValues() {
        if (this._nodeSupportsEIP1559 === undefined) {
            const block = (await this._wrappedProvider.request({
                method: "eth_getBlockByNumber",
                params: ["latest", false],
            }));
            this._nodeSupportsEIP1559 = block.baseFeePerGas !== undefined;
        }
        if (this._nodeHasFeeHistory === false ||
            this._nodeSupportsEIP1559 === false) {
            return;
        }
        try {
            const response = (await this._wrappedProvider.request({
                method: "eth_feeHistory",
                params: [
                    "0x1",
                    "latest",
                    [AutomaticGasPriceProvider.EIP1559_REWARD_PERCENTILE],
                ],
            }));
            let maxPriorityFeePerGas = (0, base_types_1.rpcQuantityToBigInt)(response.reward[0][0]);
            if (maxPriorityFeePerGas === 0n) {
                try {
                    const suggestedMaxPriorityFeePerGas = (await this._wrappedProvider.request({
                        method: "eth_maxPriorityFeePerGas",
                        params: [],
                    }));
                    maxPriorityFeePerGas = (0, base_types_1.rpcQuantityToBigInt)(suggestedMaxPriorityFeePerGas);
                }
                catch {
                    // if eth_maxPriorityFeePerGas does not exist, use 1 wei
                    maxPriorityFeePerGas = 1n;
                }
            }
            // If after all of these we still have a 0 wei maxPriorityFeePerGas, we
            // use 1 wei. This is to improve the UX of the automatic gas price
            // on chains that are very empty (i.e local testnets). This will be very
            // unlikely to trigger on a live chain.
            if (maxPriorityFeePerGas === 0n) {
                maxPriorityFeePerGas = 1n;
            }
            return {
                // Each block increases the base fee by 1/8 at most, when full.
                // We have the next block's base fee, so we compute a cap for the
                // next N blocks here.
                maxFeePerGas: ((0, base_types_1.rpcQuantityToBigInt)(response.baseFeePerGas[1]) *
                    9n **
                        (AutomaticGasPriceProvider.EIP1559_BASE_FEE_MAX_FULL_BLOCKS_PREFERENCE -
                            1n)) /
                    8n **
                        (AutomaticGasPriceProvider.EIP1559_BASE_FEE_MAX_FULL_BLOCKS_PREFERENCE -
                            1n),
                maxPriorityFeePerGas,
            };
        }
        catch {
            this._nodeHasFeeHistory = false;
            return undefined;
        }
    }
}
// We pay the max base fee that can be required if the next
// EIP1559_BASE_FEE_MAX_FULL_BLOCKS_PREFERENCE are full.
AutomaticGasPriceProvider.EIP1559_BASE_FEE_MAX_FULL_BLOCKS_PREFERENCE = 3n;
// See eth_feeHistory for an explanation of what this means
AutomaticGasPriceProvider.EIP1559_REWARD_PERCENTILE = 50;
exports.AutomaticGasPriceProvider = AutomaticGasPriceProvider;
//# sourceMappingURL=gas-providers.js.map