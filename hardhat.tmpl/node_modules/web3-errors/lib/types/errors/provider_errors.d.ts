import { BaseWeb3Error } from '../web3_error_base.js';
export declare class ProviderError extends BaseWeb3Error {
    code: number;
}
export declare class InvalidProviderError extends BaseWeb3Error {
    clientUrl: string;
    code: number;
    constructor(clientUrl: string);
}
export declare class InvalidClientError extends BaseWeb3Error {
    code: number;
    constructor(clientUrl: string);
}
export declare class SubscriptionError extends BaseWeb3Error {
    code: number;
}
export declare class Web3WSProviderError extends BaseWeb3Error {
    code: number;
}
//# sourceMappingURL=provider_errors.d.ts.map