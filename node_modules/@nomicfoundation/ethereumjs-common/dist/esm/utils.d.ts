declare type ConfigHardfork = {
    name: string;
    block: null;
    timestamp: number;
} | {
    name: string;
    block: number;
    timestamp?: number;
};
/**
 * Parses a genesis.json exported from Geth into parameters for Common instance
 * @param json representing the Geth genesis file
 * @param name optional chain name
 * @returns parsed params
 */
export declare function parseGethGenesis(json: any, name?: string, mergeForkIdPostMerge?: boolean): {
    name: string;
    chainId: number;
    networkId: number;
    genesis: {
        timestamp: string;
        gasLimit: string;
        difficulty: string;
        nonce: string;
        extraData: string;
        mixHash: string;
        coinbase: string;
        baseFeePerGas: string;
        excessBlobGas: string;
    };
    hardfork: string | undefined;
    hardforks: ConfigHardfork[];
    bootstrapNodes: never[];
    consensus: {
        type: string;
        algorithm: string;
        clique: {
            period: any;
            epoch: any;
        };
        ethash?: undefined;
    } | {
        type: string;
        algorithm: string;
        ethash: {};
        clique?: undefined;
    };
};
export {};
//# sourceMappingURL=utils.d.ts.map