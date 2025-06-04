import { type IWeb3Provider } from '../utils.ts';
export type SwapOpt = {
    slippagePercent: number;
    ttl: number;
};
export declare const DEFAULT_SWAP_OPT: SwapOpt;
export type ExchangeTx = {
    address: string;
    amount: string;
    currency: string;
    expectedAmount: string;
    data?: string;
    allowance?: {
        token: string;
        contract: string;
        amount: string;
    };
    txId?: string;
};
export type SwapElm = {
    name: string;
    expectedAmount: string;
    tx: (fromAddress: string, toAddress: string) => Promise<ExchangeTx>;
};
export declare function addPercent(n: bigint, _perc: number): bigint;
export declare function isPromise(o: unknown): boolean;
export type UnPromise<T> = T extends Promise<infer U> ? U : T;
type NestedUnPromise<T> = {
    [K in keyof T]: NestedUnPromise<UnPromise<T[K]>>;
};
type UnPromiseIgnore<T> = T extends Promise<infer U> ? U | undefined : T;
type NestedUnPromiseIgnore<T> = {
    [K in keyof T]: NestedUnPromiseIgnore<UnPromiseIgnore<T[K]>>;
};
export declare function awaitDeep<T, E extends boolean | undefined>(o: T, ignore_errors: E): Promise<E extends true ? NestedUnPromiseIgnore<T> : NestedUnPromise<T>>;
export type CommonBase = {
    contract: string;
} & import('../abi/decoder.js').ContractInfo;
export declare const COMMON_BASES: CommonBase[];
export declare const WETH: string;
export declare function wrapContract(contract: string): string;
export declare function sortTokens(a: string, b: string): [string, string];
export declare function isValidEthAddr(address: string): boolean;
export declare function isValidUniAddr(address: string): boolean;
export type Token = {
    decimals: number;
    contract: string;
    symbol: string;
};
export declare abstract class UniswapAbstract {
    abstract name: string;
    abstract contract: string;
    abstract bestPath(fromCoin: string, toCoin: string, inputAmount: bigint): any;
    abstract txData(toAddress: string, fromCoin: string, toCoin: string, path: any, inputAmount?: bigint, outputAmount?: bigint, opt?: {
        slippagePercent: number;
    }): any;
    readonly net: IWeb3Provider;
    constructor(net: IWeb3Provider);
    swap(fromCoin: 'eth' | Token, toCoin: 'eth' | Token, amount: string, opt?: SwapOpt): Promise<{
        name: string;
        expectedAmount: string;
        tx: (_fromAddress: string, toAddress: string) => Promise<{
            amount: string;
            address: any;
            expectedAmount: string;
            data: string;
            allowance: any;
        }>;
    } | undefined>;
}
export {};
//# sourceMappingURL=uniswap-common.d.ts.map