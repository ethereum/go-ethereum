import * as uni from './uniswap-common.ts';
export declare function create2(from: Uint8Array, salt: Uint8Array, initCodeHash: Uint8Array): string;
export declare function pairAddress(a: string, b: string, factory?: string): string;
export declare function amount(reserveIn: bigint, reserveOut: bigint, amountIn?: bigint, amountOut?: bigint): bigint;
export type Path = {
    path: string[];
    amountIn: bigint;
    amountOut: bigint;
};
export declare function txData(to: string, input: string, output: string, path: Path, amountIn?: bigint, amountOut?: bigint, opt?: {
    ttl: number;
    deadline?: number;
    slippagePercent: number;
    feeOnTransfer: boolean;
}): {
    to: string;
    value: bigint;
    data: any;
    allowance: {
        token: string;
        amount: bigint;
    } | undefined;
};
export default class UniswapV2 extends uni.UniswapAbstract {
    name: string;
    contract: string;
    bestPath(fromCoin: string, toCoin: string, inputAmount: bigint): Promise<Path>;
    txData(toAddress: string, fromCoin: string, toCoin: string, path: any, inputAmount?: bigint, outputAmount?: bigint, opt?: uni.SwapOpt): any;
}
//# sourceMappingURL=uniswap-v2.d.ts.map