"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.JsonRpcBatchProvider = void 0;
var properties_1 = require("@ethersproject/properties");
var web_1 = require("@ethersproject/web");
var json_rpc_provider_1 = require("./json-rpc-provider");
// Experimental
var JsonRpcBatchProvider = /** @class */ (function (_super) {
    __extends(JsonRpcBatchProvider, _super);
    function JsonRpcBatchProvider() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    JsonRpcBatchProvider.prototype.send = function (method, params) {
        var _this = this;
        var request = {
            method: method,
            params: params,
            id: (this._nextId++),
            jsonrpc: "2.0"
        };
        if (this._pendingBatch == null) {
            this._pendingBatch = [];
        }
        var inflightRequest = { request: request, resolve: null, reject: null };
        var promise = new Promise(function (resolve, reject) {
            inflightRequest.resolve = resolve;
            inflightRequest.reject = reject;
        });
        this._pendingBatch.push(inflightRequest);
        if (!this._pendingBatchAggregator) {
            // Schedule batch for next event loop + short duration
            this._pendingBatchAggregator = setTimeout(function () {
                // Get teh current batch and clear it, so new requests
                // go into the next batch
                var batch = _this._pendingBatch;
                _this._pendingBatch = null;
                _this._pendingBatchAggregator = null;
                // Get the request as an array of requests
                var request = batch.map(function (inflight) { return inflight.request; });
                _this.emit("debug", {
                    action: "requestBatch",
                    request: (0, properties_1.deepCopy)(request),
                    provider: _this
                });
                return (0, web_1.fetchJson)(_this.connection, JSON.stringify(request)).then(function (result) {
                    _this.emit("debug", {
                        action: "response",
                        request: request,
                        response: result,
                        provider: _this
                    });
                    // For each result, feed it to the correct Promise, depending
                    // on whether it was a success or error
                    batch.forEach(function (inflightRequest, index) {
                        var payload = result[index];
                        if (payload.error) {
                            var error = new Error(payload.error.message);
                            error.code = payload.error.code;
                            error.data = payload.error.data;
                            inflightRequest.reject(error);
                        }
                        else {
                            inflightRequest.resolve(payload.result);
                        }
                    });
                }, function (error) {
                    _this.emit("debug", {
                        action: "response",
                        error: error,
                        request: request,
                        provider: _this
                    });
                    batch.forEach(function (inflightRequest) {
                        inflightRequest.reject(error);
                    });
                });
            }, 10);
        }
        return promise;
    };
    return JsonRpcBatchProvider;
}(json_rpc_provider_1.JsonRpcProvider));
exports.JsonRpcBatchProvider = JsonRpcBatchProvider;
//# sourceMappingURL=json-rpc-batch-provider.js.map