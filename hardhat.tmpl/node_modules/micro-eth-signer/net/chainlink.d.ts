import { type IWeb3Provider } from '../utils.ts';
export declare const TOKENS: Record<string, {
    decimals: number;
    contract: string;
    tokenContract: string;
}>;
export default class Chainlink {
    readonly net: IWeb3Provider;
    constructor(net: IWeb3Provider);
    price(contract: string, decimals: number): Promise<number>;
    coinPrice(symbol: string): Promise<number>;
    tokenPrice(symbol: string): Promise<number>;
}
//# sourceMappingURL=chainlink.d.ts.map