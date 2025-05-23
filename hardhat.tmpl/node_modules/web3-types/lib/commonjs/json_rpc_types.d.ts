export type JsonRpcId = string | number | undefined;
export type JsonRpcResult = string | number | boolean | Record<string, unknown>;
export type JsonRpcIdentifier = string & ('2.0' | '1.0');
export interface JsonRpcError<T = JsonRpcResult> {
    readonly code: number;
    readonly message: string;
    readonly data?: T;
}
export interface JsonRpcResponseWithError<Error = JsonRpcResult> {
    readonly id: JsonRpcId;
    readonly jsonrpc: JsonRpcIdentifier;
    readonly error: JsonRpcError<Error>;
    readonly result?: never;
}
export interface JsonRpcResponseWithResult<T = JsonRpcResult> {
    readonly id: JsonRpcId;
    readonly jsonrpc: JsonRpcIdentifier;
    readonly error?: never;
    readonly result: T;
}
export interface SubscriptionParams<T = JsonRpcResult> {
    readonly subscription: string;
    readonly result: T;
}
export interface JsonRpcSubscriptionResultOld<T = JsonRpcResult> {
    readonly error?: never;
    readonly params?: never;
    readonly type: string;
    readonly data: SubscriptionParams<T>;
}
export interface JsonRpcNotification<T = JsonRpcResult> {
    readonly id?: JsonRpcId;
    readonly jsonrpc: JsonRpcIdentifier;
    readonly method: string;
    readonly params: SubscriptionParams<T>;
    readonly result?: never;
    readonly data?: never;
    readonly error?: never;
}
export interface JsonRpcSubscriptionResult {
    readonly id: number;
    readonly jsonrpc: string;
    readonly result: string;
    readonly method: never;
    readonly params: never;
    readonly data?: never;
}
export interface JsonRpcRequest<T = unknown[]> {
    readonly id: JsonRpcId;
    readonly jsonrpc: JsonRpcIdentifier;
    readonly method: string;
    readonly params?: T;
}
export interface JsonRpcOptionalRequest<ParamType = unknown[]> extends Omit<JsonRpcRequest<ParamType>, 'id' | 'jsonrpc'> {
    readonly id?: JsonRpcId;
    readonly jsonrpc?: JsonRpcIdentifier;
}
export type JsonRpcBatchRequest = JsonRpcRequest[];
export type JsonRpcPayload<Param = unknown[]> = JsonRpcRequest<Param> | JsonRpcBatchRequest;
export type JsonRpcBatchResponse<Result = JsonRpcResult, Error = JsonRpcResult> = (JsonRpcResponseWithError<Error> | JsonRpcResponseWithResult<Result>)[];
export type JsonRpcResponse<Result = JsonRpcResult, Error = JsonRpcResult> = JsonRpcResponseWithError<Error> | JsonRpcResponseWithResult<Result> | JsonRpcBatchResponse<Result, Error> | JsonRpcNotification<Result>;
