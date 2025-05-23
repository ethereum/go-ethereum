import { deepCopy } from "@ethersproject/properties";
import { fetchJson } from "@ethersproject/web";
import { JsonRpcProvider } from "./json-rpc-provider";
// Experimental
export class JsonRpcBatchProvider extends JsonRpcProvider {
    send(method, params) {
        const request = {
            method: method,
            params: params,
            id: (this._nextId++),
            jsonrpc: "2.0"
        };
        if (this._pendingBatch == null) {
            this._pendingBatch = [];
        }
        const inflightRequest = { request, resolve: null, reject: null };
        const promise = new Promise((resolve, reject) => {
            inflightRequest.resolve = resolve;
            inflightRequest.reject = reject;
        });
        this._pendingBatch.push(inflightRequest);
        if (!this._pendingBatchAggregator) {
            // Schedule batch for next event loop + short duration
            this._pendingBatchAggregator = setTimeout(() => {
                // Get teh current batch and clear it, so new requests
                // go into the next batch
                const batch = this._pendingBatch;
                this._pendingBatch = null;
                this._pendingBatchAggregator = null;
                // Get the request as an array of requests
                const request = batch.map((inflight) => inflight.request);
                this.emit("debug", {
                    action: "requestBatch",
                    request: deepCopy(request),
                    provider: this
                });
                return fetchJson(this.connection, JSON.stringify(request)).then((result) => {
                    this.emit("debug", {
                        action: "response",
                        request: request,
                        response: result,
                        provider: this
                    });
                    // For each result, feed it to the correct Promise, depending
                    // on whether it was a success or error
                    batch.forEach((inflightRequest, index) => {
                        const payload = result[index];
                        if (payload.error) {
                            const error = new Error(payload.error.message);
                            error.code = payload.error.code;
                            error.data = payload.error.data;
                            inflightRequest.reject(error);
                        }
                        else {
                            inflightRequest.resolve(payload.result);
                        }
                    });
                }, (error) => {
                    this.emit("debug", {
                        action: "response",
                        error: error,
                        request: request,
                        provider: this
                    });
                    batch.forEach((inflightRequest) => {
                        inflightRequest.reject(error);
                    });
                });
            }, 10);
        }
        return promise;
    }
}
//# sourceMappingURL=json-rpc-batch-provider.js.map