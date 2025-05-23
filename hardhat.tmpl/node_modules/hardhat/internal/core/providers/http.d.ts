/// <reference types="node" />
import type * as Undici from "undici";
import { EventEmitter } from "events";
import { EIP1193Provider, RequestArguments } from "../../../types";
import { FailedJsonRpcResponse } from "../../util/jsonrpc";
export declare function isErrorResponse(response: any): response is FailedJsonRpcResponse;
export declare class HttpProvider extends EventEmitter implements EIP1193Provider {
    private readonly _url;
    private readonly _networkName;
    private readonly _extraHeaders;
    private readonly _timeout;
    private _nextRequestId;
    private _dispatcher;
    private _path;
    private _authHeader;
    constructor(_url: string, _networkName: string, _extraHeaders?: {
        [name: string]: string;
    }, _timeout?: number, client?: Undici.Dispatcher | undefined);
    get url(): string;
    request(args: RequestArguments): Promise<unknown>;
    /**
     * Sends a batch of requests. Fails if any of them fails.
     */
    sendBatch(batch: Array<{
        method: string;
        params: any[];
    }>): Promise<any[]>;
    private _fetchJsonRpcResponse;
    private _retry;
    private _getJsonRpcRequest;
    private _shouldRetry;
    private _isRateLimitResponse;
    private _getRetryAfterSeconds;
}
//# sourceMappingURL=http.d.ts.map