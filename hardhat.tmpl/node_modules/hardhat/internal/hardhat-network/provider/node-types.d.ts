import { HARDHAT_MEMPOOL_SUPPORTED_ORDERS } from "../../constants";
export interface ForkConfig {
    jsonRpcUrl: string;
    blockNumber?: number;
    httpHeaders?: {
        [name: string]: string;
    };
}
export type IntervalMiningConfig = number | [number, number];
export type MempoolOrder = typeof HARDHAT_MEMPOOL_SUPPORTED_ORDERS[number];
export interface GenesisAccount {
    privateKey: string;
    balance: string | number | bigint;
}
//# sourceMappingURL=node-types.d.ts.map