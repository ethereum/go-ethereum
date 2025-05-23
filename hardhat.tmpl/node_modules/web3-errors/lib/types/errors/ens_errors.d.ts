import { BaseWeb3Error } from '../web3_error_base.js';
export declare class ENSCheckInterfaceSupportError extends BaseWeb3Error {
    code: number;
    constructor(errorDetails: string);
}
export declare class ENSUnsupportedNetworkError extends BaseWeb3Error {
    code: number;
    constructor(networkType: string);
}
export declare class ENSNetworkNotSyncedError extends BaseWeb3Error {
    code: number;
    constructor();
}
//# sourceMappingURL=ens_errors.d.ts.map