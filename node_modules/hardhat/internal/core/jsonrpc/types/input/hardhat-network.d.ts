import * as t from "io-ts";
export declare const rpcForkConfig: t.Type<{
    jsonRpcUrl: string;
    blockNumber: number | undefined;
    httpHeaders: {
        [x: string]: string;
    } | undefined;
} | undefined, {
    jsonRpcUrl: string;
    blockNumber: number | undefined;
    httpHeaders: {
        [x: string]: string;
    } | undefined;
} | undefined, unknown>;
export type RpcForkConfig = t.TypeOf<typeof rpcForkConfig>;
export declare const rpcHardhatNetworkConfig: t.TypeC<{
    forking: t.Type<{
        jsonRpcUrl: string;
        blockNumber: number | undefined;
        httpHeaders: {
            [x: string]: string;
        } | undefined;
    } | undefined, {
        jsonRpcUrl: string;
        blockNumber: number | undefined;
        httpHeaders: {
            [x: string]: string;
        } | undefined;
    } | undefined, unknown>;
}>;
export type RpcHardhatNetworkConfig = t.TypeOf<typeof rpcHardhatNetworkConfig>;
export declare const optionalRpcHardhatNetworkConfig: t.Type<{
    forking: {
        jsonRpcUrl: string;
        blockNumber: number | undefined;
        httpHeaders: {
            [x: string]: string;
        } | undefined;
    } | undefined;
} | undefined, {
    forking: {
        jsonRpcUrl: string;
        blockNumber: number | undefined;
        httpHeaders: {
            [x: string]: string;
        } | undefined;
    } | undefined;
} | undefined, unknown>;
export declare const rpcIntervalMining: t.UnionC<[t.Type<number, number, unknown>, t.Type<[number, number], [number, number], unknown>]>;
export type RpcIntervalMining = t.TypeOf<typeof rpcIntervalMining>;
//# sourceMappingURL=hardhat-network.d.ts.map