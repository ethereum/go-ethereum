import { ProviderRpcError } from "../../../types";
import { CustomError } from "../errors";
export declare class ProviderError extends CustomError implements ProviderRpcError {
    readonly parent?: Error | undefined;
    static isProviderError(other: any): other is ProviderError;
    code: number;
    data?: unknown;
    private readonly _isProviderError;
    constructor(message: string, code: number, parent?: Error | undefined);
}
export declare class InvalidJsonInputError extends ProviderError {
    static readonly CODE = -32700;
    constructor(message: string, parent?: Error);
}
export declare class InvalidRequestError extends ProviderError {
    static readonly CODE = -32600;
    constructor(message: string, parent?: Error);
}
export declare class MethodNotFoundError extends ProviderError {
    static readonly CODE = -32601;
    constructor(message: string, parent?: Error);
}
export declare class InvalidArgumentsError extends ProviderError {
    static readonly CODE = -32602;
    constructor(message: string, parent?: Error);
}
export declare class InternalError extends ProviderError {
    static readonly CODE = -32603;
    constructor(message: string, parent?: Error);
}
export declare class InvalidInputError extends ProviderError {
    static readonly CODE = -32000;
    constructor(message: string, parent?: Error);
}
export declare class TransactionExecutionError extends ProviderError {
    static readonly CODE = -32003;
    constructor(parentOrMsg: Error | string);
}
export declare class MethodNotSupportedError extends ProviderError {
    static readonly CODE = -32004;
    constructor(method: string, parent?: Error);
}
export declare class InvalidResponseError extends ProviderError {
    static readonly CODE = -32999;
    constructor(message: string, parent?: Error);
}
//# sourceMappingURL=errors.d.ts.map