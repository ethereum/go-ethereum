import { Web3Error } from 'web3-types';
/**
 * Base class for Web3 errors.
 */
export declare abstract class BaseWeb3Error extends Error implements Web3Error {
    readonly name: string;
    abstract readonly code: number;
    stack: string | undefined;
    cause: Error | undefined;
    /**
     * @deprecated Use the `cause` property instead.
     */
    get innerError(): Error | Error[] | undefined;
    /**
     * @deprecated Use the `cause` property instead.
     */
    set innerError(cause: Error | Error[] | undefined);
    constructor(msg?: string, cause?: Error | Error[]);
    static convertToString(value: unknown, unquotValue?: boolean): string;
    toJSON(): {
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class MultipleErrors extends BaseWeb3Error {
    code: number;
    errors: Error[];
    constructor(errors: Error[]);
}
export declare abstract class InvalidValueError extends BaseWeb3Error {
    readonly name: string;
    constructor(value: unknown, msg: string);
}
//# sourceMappingURL=web3_error_base.d.ts.map