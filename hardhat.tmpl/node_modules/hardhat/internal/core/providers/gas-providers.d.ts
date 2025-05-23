import { EIP1193Provider, RequestArguments } from "../../../types";
import { ProviderWrapper } from "./wrapper";
export declare class FixedGasProvider extends ProviderWrapper {
    private readonly _gasLimit;
    constructor(provider: EIP1193Provider, _gasLimit: number);
    request(args: RequestArguments): Promise<unknown>;
}
export declare class FixedGasPriceProvider extends ProviderWrapper {
    private readonly _gasPrice;
    constructor(provider: EIP1193Provider, _gasPrice: number);
    request(args: RequestArguments): Promise<unknown>;
}
declare abstract class MultipliedGasEstimationProvider extends ProviderWrapper {
    private readonly _gasMultiplier;
    private _blockGasLimit;
    constructor(provider: EIP1193Provider, _gasMultiplier: number);
    protected _getMultipliedGasEstimation(params: any[]): Promise<string>;
    private _getBlockGasLimit;
}
export declare class AutomaticGasProvider extends MultipliedGasEstimationProvider {
    constructor(provider: EIP1193Provider, gasMultiplier?: number);
    request(args: RequestArguments): Promise<unknown>;
}
export declare class AutomaticGasPriceProvider extends ProviderWrapper {
    static readonly EIP1559_BASE_FEE_MAX_FULL_BLOCKS_PREFERENCE: bigint;
    static readonly EIP1559_REWARD_PERCENTILE = 50;
    private _nodeHasFeeHistory?;
    private _nodeSupportsEIP1559?;
    request(args: RequestArguments): Promise<unknown>;
    private _getGasPrice;
    private _suggestEip1559FeePriceValues;
}
export {};
//# sourceMappingURL=gas-providers.d.ts.map