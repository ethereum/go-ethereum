/// <reference types="node" />
import type { Artifacts, EIP1193Provider, HardhatNetworkChainsConfig, RequestArguments } from "../../../types";
import type { EdrContext, TracingConfigWithBuffers } from "@nomicfoundation/edr";
import { EventEmitter } from "events";
import { ForkConfig, GenesisAccount, IntervalMiningConfig, MempoolOrder } from "./node-types";
import { LoggerConfig } from "./modules/logger";
export declare const DEFAULT_COINBASE = "0xc014ba5ec014ba5ec014ba5ec014ba5ec014ba5e";
export declare function getGlobalEdrContext(): EdrContext;
interface HardhatNetworkProviderConfig {
    hardfork: string;
    chainId: number;
    networkId: number;
    blockGasLimit: number;
    minGasPrice: bigint;
    automine: boolean;
    intervalMining: IntervalMiningConfig;
    mempoolOrder: MempoolOrder;
    chains: HardhatNetworkChainsConfig;
    genesisAccounts: GenesisAccount[];
    allowUnlimitedContractSize: boolean;
    throwOnTransactionFailures: boolean;
    throwOnCallFailures: boolean;
    allowBlocksWithSameTimestamp: boolean;
    initialBaseFeePerGas?: number;
    initialDate?: Date;
    coinbase?: string;
    forkConfig?: ForkConfig;
    forkCachePath?: string;
    enableTransientStorage: boolean;
    enableRip7212: boolean;
}
export declare class EdrProviderWrapper extends EventEmitter implements EIP1193Provider {
    private readonly _provider;
    private readonly _node;
    private _failedStackTraces;
    private _callOverrideCallback?;
    private constructor();
    static create(config: HardhatNetworkProviderConfig, loggerConfig: LoggerConfig, tracingConfig?: TracingConfigWithBuffers): Promise<EdrProviderWrapper>;
    request(args: RequestArguments): Promise<unknown>;
    private _setCallOverrideCallback;
    private _setVerboseTracing;
    private _ethEventListener;
    private _emitLegacySubscriptionEvent;
    private _emitEip1193SubscriptionEvent;
}
export declare function createHardhatNetworkProvider(hardhatNetworkProviderConfig: HardhatNetworkProviderConfig, loggerConfig: LoggerConfig, artifacts?: Artifacts): Promise<EIP1193Provider>;
export {};
//# sourceMappingURL=provider.d.ts.map