import { JsonRpcResponseWithError, JsonRpcId, JsonRpcError } from 'web3-types';
import { BaseWeb3Error } from '../web3_error_base.js';
export declare class RpcError extends BaseWeb3Error {
    code: number;
    id: JsonRpcId;
    jsonrpc: string;
    jsonRpcError: JsonRpcError;
    constructor(rpcError: JsonRpcResponseWithError, message?: string);
    toJSON(): {
        error: JsonRpcError<import("web3-types").JsonRpcResult>;
        id: JsonRpcId;
        jsonRpc: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class EIP1193ProviderRpcError extends BaseWeb3Error {
    code: number;
    data?: unknown;
    constructor(code: number, data?: unknown);
}
export declare class ParseError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class InvalidRequestError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class MethodNotFoundError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class InvalidParamsError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class InternalError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class InvalidInputError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class MethodNotSupported extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class ResourceUnavailableError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class ResourcesNotFoundError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class VersionNotSupportedError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class TransactionRejectedError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare class LimitExceededError extends RpcError {
    code: number;
    constructor(rpcError: JsonRpcResponseWithError);
}
export declare const rpcErrorsMap: Map<number, {
    error: typeof RpcError;
}>;
