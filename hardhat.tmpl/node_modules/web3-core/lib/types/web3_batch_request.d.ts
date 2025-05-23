import { JsonRpcBatchResponse, JsonRpcOptionalRequest, JsonRpcRequest } from 'web3-types';
import { Web3DeferredPromise } from 'web3-utils';
import { Web3RequestManager } from './web3_request_manager.js';
export declare const DEFAULT_BATCH_REQUEST_TIMEOUT = 1000;
export declare class Web3BatchRequest {
    private readonly _requestManager;
    private readonly _requests;
    constructor(requestManager: Web3RequestManager);
    get requests(): JsonRpcRequest<unknown[]>[];
    add<ResponseType = unknown>(request: JsonRpcOptionalRequest<unknown>): Web3DeferredPromise<ResponseType>;
    execute(options?: {
        timeout?: number;
    }): Promise<JsonRpcBatchResponse<unknown, unknown>>;
    private _processBatchRequest;
    private _abortAllRequests;
}
//# sourceMappingURL=web3_batch_request.d.ts.map