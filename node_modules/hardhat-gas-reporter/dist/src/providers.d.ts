import { ProviderWrapper } from "hardhat/internal/core/providers/wrapper";
import { EthereumProvider, EIP1193Provider, RequestArguments } from "hardhat/types";
/**
 * Wrapped provider which collects tx data
 */
export declare class EGRDataCollectionProvider extends ProviderWrapper {
    private mochaConfig;
    constructor(provider: EIP1193Provider, mochaConfig: any);
    request(args: RequestArguments): Promise<unknown>;
}
/**
 * A set of async RPC calls which substitute the sync calls made by the core reporter
 * These allow us to use HardhatEVM or another in-process provider.
 */
export declare class EGRAsyncApiProvider {
    provider: EthereumProvider;
    constructor(provider: EthereumProvider);
    getNetworkId(): Promise<any>;
    getCode(address: string): Promise<any>;
    getLatestBlock(): Promise<any>;
    getBlockByNumber(num: number): Promise<any>;
    blockNumber(): Promise<number>;
    getTransactionByHash(tx: any): Promise<any>;
    call(payload: any, blockNumber: any): Promise<any>;
}
//# sourceMappingURL=providers.d.ts.map