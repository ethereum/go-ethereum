import { BaseWeb3Error } from 'web3-errors';
export declare class QuickNodeRateLimitError extends BaseWeb3Error {
    code: number;
    constructor(error?: Error);
}
export declare class ProviderConfigOptionsError extends BaseWeb3Error {
    code: number;
    constructor(msg: string);
}
//# sourceMappingURL=errors.d.ts.map