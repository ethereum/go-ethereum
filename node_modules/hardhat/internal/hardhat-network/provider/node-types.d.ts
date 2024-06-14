import { HARDHAT_MEMPOOL_SUPPORTED_ORDERS } from "../../constants";
import { BuildInfo, HardhatNetworkChainsConfig } from "../../../types";
export type NodeConfig = LocalNodeConfig | ForkedNodeConfig;
interface CommonConfig {
    automine: boolean;
    blockGasLimit: number;
    chainId: number;
    genesisAccounts: GenesisAccount[];
    hardfork: string;
    minGasPrice: bigint;
    networkId: number;
    allowUnlimitedContractSize?: boolean;
    initialDate?: Date;
    tracingConfig?: TracingConfig;
    initialBaseFeePerGas?: number;
    mempoolOrder: MempoolOrder;
    coinbase: string;
    chains: HardhatNetworkChainsConfig;
    allowBlocksWithSameTimestamp: boolean;
    enableTransientStorage: boolean;
}
export type LocalNodeConfig = CommonConfig;
export interface ForkConfig {
    jsonRpcUrl: string;
    blockNumber?: number;
    httpHeaders?: {
        [name: string]: string;
    };
}
export interface ForkedNodeConfig extends CommonConfig {
    forkConfig: ForkConfig;
    forkCachePath?: string;
}
export interface TracingConfig {
    buildInfos?: BuildInfo[];
    ignoreContracts?: boolean;
}
export type IntervalMiningConfig = number | [number, number];
export type MempoolOrder = typeof HARDHAT_MEMPOOL_SUPPORTED_ORDERS[number];
export interface GenesisAccount {
    privateKey: string;
    balance: string | number | bigint;
}
export type AccessListBufferItem = [Uint8Array, Uint8Array[]];
export type TransactionParams = LegacyTransactionParams | AccessListTransactionParams | EIP1559TransactionParams;
interface BaseTransactionParams {
    to?: Uint8Array;
    from: Uint8Array;
    gasLimit: bigint;
    value: bigint;
    data: Uint8Array;
    nonce: bigint;
}
export interface LegacyTransactionParams extends BaseTransactionParams {
    gasPrice: bigint;
}
export interface AccessListTransactionParams extends BaseTransactionParams {
    gasPrice: bigint;
    accessList: AccessListBufferItem[];
}
export interface EIP1559TransactionParams extends BaseTransactionParams {
    accessList: AccessListBufferItem[];
    maxFeePerGas: bigint;
    maxPriorityFeePerGas: bigint;
}
export {};
//# sourceMappingURL=node-types.d.ts.map