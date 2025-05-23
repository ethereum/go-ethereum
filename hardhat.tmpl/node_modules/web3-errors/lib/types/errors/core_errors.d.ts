import { BaseWeb3Error } from '../web3_error_base.js';
export declare class ConfigHardforkMismatchError extends BaseWeb3Error {
    code: number;
    constructor(defaultHardfork: string, commonHardFork: string);
}
export declare class ConfigChainMismatchError extends BaseWeb3Error {
    code: number;
    constructor(defaultHardfork: string, commonHardFork: string);
}
//# sourceMappingURL=core_errors.d.ts.map