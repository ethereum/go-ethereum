import { BaseWeb3Error } from '../web3_error_base.js';
export declare class InvalidNumberOfParamsError extends BaseWeb3Error {
    got: number;
    expected: number;
    method: string;
    code: number;
    constructor(got: number, expected: number, method: string);
    toJSON(): {
        got: number;
        expected: number;
        method: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class InvalidMethodParamsError extends BaseWeb3Error {
    hint?: string | undefined;
    code: number;
    constructor(hint?: string | undefined);
    toJSON(): {
        hint: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class FormatterError extends BaseWeb3Error {
    code: number;
}
export declare class MethodNotImplementedError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class OperationTimeoutError extends BaseWeb3Error {
    code: number;
}
export declare class OperationAbortError extends BaseWeb3Error {
    code: number;
}
export declare class AbiError extends BaseWeb3Error {
    code: number;
    readonly props: Record<string, unknown> & {
        name?: string;
    };
    constructor(message: string, props?: Record<string, unknown> & {
        name?: string;
    });
}
export declare class ExistingPluginNamespaceError extends BaseWeb3Error {
    code: number;
    constructor(pluginNamespace: string);
}
