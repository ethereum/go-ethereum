import * as uni from './uniswap-common.ts';
export declare const Fee: Record<string, number>;
type Route = {
    path?: Uint8Array;
    fee?: number;
    amountIn?: bigint;
    amountOut?: bigint;
    p?: any;
};
export type TxOpt = {
    slippagePercent: number;
    ttl: number;
    sqrtPriceLimitX96?: bigint;
    deadline?: number;
    fee?: {
        fee: number;
        to: string;
    };
};
export declare function txData(to: string, input: string, output: string, route: Route, amountIn?: bigint, amountOut?: bigint, opt?: TxOpt): {
    to: string;
    value: bigint;
    data: Uint8Array;
    allowance: {
        token: string;
        amount: bigint;
    } | undefined;
};
export default class UniswapV3 extends uni.UniswapAbstract {
    name: string;
    contract: string;
    bestPath(fromCoin: string, toCoin: string, inputAmount: bigint): Promise<Route>;
    txData(toAddress: string, fromCoin: string, toCoin: string, path: any, inputAmount?: bigint, outputAmount?: bigint, opt?: uni.SwapOpt): any;
}
export {};
//# sourceMappingURL=uniswap-v3.d.ts.map