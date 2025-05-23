"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SocketProvider = void 0;
const web3_errors_1 = require("web3-errors");
const web3_eip1193_provider_js_1 = require("./web3_eip1193_provider.js");
const chunk_response_parser_js_1 = require("./chunk_response_parser.js");
const validation_js_1 = require("./validation.js");
const web3_deferred_promise_js_1 = require("./web3_deferred_promise.js");
const jsonRpc = __importStar(require("./json_rpc.js"));
const DEFAULT_RECONNECTION_OPTIONS = {
    autoReconnect: true,
    delay: 5000,
    maxAttempts: 5,
};
const NORMAL_CLOSE_CODE = 1000; // https://developer.mozilla.org/en-US/docs/Web/API/WebSocket/close
class SocketProvider extends web3_eip1193_provider_js_1.Eip1193Provider {
    get SocketConnection() {
        return this._socketConnection;
    }
    /**
     * This is an abstract class for implementing a socket provider (e.g. WebSocket, IPC). It extends the EIP-1193 provider {@link EIP1193Provider}.
     * @param socketPath - The path to the socket (e.g. /ipc/path or ws://localhost:8546)
     * @param socketOptions - The options for the socket connection. Its type is supposed to be specified in the inherited classes.
     * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
     */
    constructor(socketPath, socketOptions, reconnectOptions) {
        super();
        this._connectionStatus = 'connecting';
        // Message handlers. Due to bounding of `this` and removing the listeners we have to keep it's reference.
        this._onMessageHandler = this._onMessage.bind(this);
        this._onOpenHandler = this._onConnect.bind(this);
        this._onCloseHandler = this._onCloseEvent.bind(this);
        this._onErrorHandler = this._onError.bind(this);
        if (!this._validateProviderPath(socketPath))
            throw new web3_errors_1.InvalidClientError(socketPath);
        this._socketPath = socketPath;
        this._socketOptions = socketOptions;
        this._reconnectOptions = Object.assign(Object.assign({}, DEFAULT_RECONNECTION_OPTIONS), (reconnectOptions !== null && reconnectOptions !== void 0 ? reconnectOptions : {}));
        this._pendingRequestsQueue = new Map();
        this._sentRequestsQueue = new Map();
        this._init();
        this.connect();
        this.chunkResponseParser = new chunk_response_parser_js_1.ChunkResponseParser(this._eventEmitter, this._reconnectOptions.autoReconnect);
        this.chunkResponseParser.onError(() => {
            this._clearQueues();
        });
        this.isReconnecting = false;
    }
    _init() {
        this._reconnectAttempts = 0;
    }
    /**
     * Try to establish a connection to the socket
     */
    connect() {
        try {
            this._openSocketConnection();
            this._connectionStatus = 'connecting';
            this._addSocketListeners();
        }
        catch (e) {
            if (!this.isReconnecting) {
                this._connectionStatus = 'disconnected';
                if (e && e.message) {
                    throw new web3_errors_1.ConnectionError(`Error while connecting to ${this._socketPath}. Reason: ${e.message}`);
                }
                else {
                    throw new web3_errors_1.InvalidClientError(this._socketPath);
                }
            }
            else {
                setImmediate(() => {
                    this._reconnect();
                });
            }
        }
    }
    // eslint-disable-next-line class-methods-use-this
    _validateProviderPath(path) {
        return !!path;
    }
    /**
     *
     * @returns the pendingRequestQueue size
     */
    // eslint-disable-next-line class-methods-use-this
    getPendingRequestQueueSize() {
        return this._pendingRequestsQueue.size;
    }
    /**
     *
     * @returns the sendPendingRequests size
     */
    // eslint-disable-next-line class-methods-use-this
    getSentRequestsQueueSize() {
        return this._sentRequestsQueue.size;
    }
    /**
     *
     * @returns `true` if the socket supports subscriptions
     */
    // eslint-disable-next-line class-methods-use-this
    supportsSubscriptions() {
        return true;
    }
    on(type, listener) {
        this._eventEmitter.on(type, listener);
    }
    once(type, listener) {
        this._eventEmitter.once(type, listener);
    }
    removeListener(type, listener) {
        this._eventEmitter.removeListener(type, listener);
    }
    _onDisconnect(code, data) {
        this._connectionStatus = 'disconnected';
        super._onDisconnect(code, data);
    }
    /**
     * Disconnects the socket
     * @param code - The code to be sent to the server
     * @param data - The data to be sent to the server
     */
    disconnect(code, data) {
        const disconnectCode = code !== null && code !== void 0 ? code : NORMAL_CLOSE_CODE;
        this._removeSocketListeners();
        if (this.getStatus() !== 'disconnected') {
            this._closeSocketConnection(disconnectCode, data);
        }
        this._onDisconnect(disconnectCode, data);
    }
    /**
     * Safely disconnects the socket, async and waits for request size to be 0 before disconnecting
     * @param forceDisconnect - If true, will clear queue after 5 attempts of waiting for both pending and sent queue to be 0
     * @param ms - Determines the ms of setInterval
     * @param code - The code to be sent to the server
     * @param data - The data to be sent to the server
     */
    safeDisconnect(code_1, data_1) {
        return __awaiter(this, arguments, void 0, function* (code, data, forceDisconnect = false, ms = 1000) {
            let retryAttempt = 0;
            const checkQueue = () => __awaiter(this, void 0, void 0, function* () {
                return new Promise(resolve => {
                    const interval = setInterval(() => {
                        if (forceDisconnect && retryAttempt >= 5) {
                            this.clearQueues();
                        }
                        if (this.getPendingRequestQueueSize() === 0 &&
                            this.getSentRequestsQueueSize() === 0) {
                            clearInterval(interval);
                            resolve(true);
                        }
                        retryAttempt += 1;
                    }, ms);
                });
            });
            yield checkQueue();
            this.disconnect(code, data);
        });
    }
    /**
     * Removes all listeners for the specified event type.
     * @param type - The event type to remove the listeners for
     */
    removeAllListeners(type) {
        this._eventEmitter.removeAllListeners(type);
    }
    _onError(event) {
        // do not emit error while trying to reconnect
        if (this.isReconnecting) {
            this._reconnect();
        }
        else {
            this._eventEmitter.emit('error', event);
        }
    }
    /**
     * Resets the socket, removing all listeners and pending requests
     */
    reset() {
        this._sentRequestsQueue.clear();
        this._pendingRequestsQueue.clear();
        this._init();
        this._removeSocketListeners();
        this._addSocketListeners();
    }
    _reconnect() {
        if (this.isReconnecting) {
            return;
        }
        this.isReconnecting = true;
        if (this._sentRequestsQueue.size > 0) {
            this._sentRequestsQueue.forEach((request, key) => {
                request.deferredPromise.reject(new web3_errors_1.PendingRequestsOnReconnectingError());
                this._sentRequestsQueue.delete(key);
            });
        }
        if (this._reconnectAttempts < this._reconnectOptions.maxAttempts) {
            this._reconnectAttempts += 1;
            setTimeout(() => {
                this._removeSocketListeners();
                this.connect(); // this can error out
                this.isReconnecting = false;
            }, this._reconnectOptions.delay);
        }
        else {
            this.isReconnecting = false;
            this._clearQueues();
            this._removeSocketListeners();
            this._eventEmitter.emit('error', new web3_errors_1.MaxAttemptsReachedOnReconnectingError(this._reconnectOptions.maxAttempts));
        }
    }
    /**
     *  Creates a request object to be sent to the server
     */
    request(request) {
        return __awaiter(this, void 0, void 0, function* () {
            if ((0, validation_js_1.isNullish)(this._socketConnection)) {
                throw new Error('Connection is undefined');
            }
            // if socket disconnected - open connection
            if (this.getStatus() === 'disconnected') {
                this.connect();
            }
            const requestId = jsonRpc.isBatchRequest(request)
                ? request[0].id
                : request.id;
            if (!requestId) {
                throw new web3_errors_1.Web3WSProviderError('Request Id not defined');
            }
            if (this._sentRequestsQueue.has(requestId)) {
                throw new web3_errors_1.RequestAlreadySentError(requestId);
            }
            const deferredPromise = new web3_deferred_promise_js_1.Web3DeferredPromise();
            deferredPromise.catch(error => {
                this._eventEmitter.emit('error', error);
            });
            const reqItem = {
                payload: request,
                deferredPromise,
            };
            if (this.getStatus() === 'connecting') {
                this._pendingRequestsQueue.set(requestId, reqItem);
                return reqItem.deferredPromise;
            }
            this._sentRequestsQueue.set(requestId, reqItem);
            try {
                this._sendToSocket(reqItem.payload);
            }
            catch (error) {
                this._sentRequestsQueue.delete(requestId);
                this._eventEmitter.emit('error', error);
            }
            return deferredPromise;
        });
    }
    _onConnect() {
        this._connectionStatus = 'connected';
        this._reconnectAttempts = 0;
        super._onConnect();
        this._sendPendingRequests();
    }
    _sendPendingRequests() {
        for (const [id, value] of this._pendingRequestsQueue.entries()) {
            try {
                this._sendToSocket(value.payload);
                this._pendingRequestsQueue.delete(id);
                this._sentRequestsQueue.set(id, value);
            }
            catch (error) {
                // catches if sendTosocket fails
                this._pendingRequestsQueue.delete(id);
                this._eventEmitter.emit('error', error);
            }
        }
    }
    _onMessage(event) {
        const responses = this._parseResponses(event);
        if ((0, validation_js_1.isNullish)(responses) || responses.length === 0) {
            return;
        }
        for (const response of responses) {
            if (jsonRpc.isResponseWithNotification(response) &&
                response.method.endsWith('_subscription')) {
                this._eventEmitter.emit('message', response);
                return;
            }
            const requestId = jsonRpc.isBatchResponse(response)
                ? response[0].id
                : response.id;
            const requestItem = this._sentRequestsQueue.get(requestId);
            if (!requestItem) {
                return;
            }
            if (jsonRpc.isBatchResponse(response) ||
                jsonRpc.isResponseWithResult(response) ||
                jsonRpc.isResponseWithError(response)) {
                this._eventEmitter.emit('message', response);
                requestItem.deferredPromise.resolve(response);
            }
            this._sentRequestsQueue.delete(requestId);
        }
    }
    clearQueues(event) {
        this._clearQueues(event);
    }
    _clearQueues(event) {
        if (this._pendingRequestsQueue.size > 0) {
            this._pendingRequestsQueue.forEach((request, key) => {
                request.deferredPromise.reject(new web3_errors_1.ConnectionNotOpenError(event));
                this._pendingRequestsQueue.delete(key);
            });
        }
        if (this._sentRequestsQueue.size > 0) {
            this._sentRequestsQueue.forEach((request, key) => {
                request.deferredPromise.reject(new web3_errors_1.ConnectionNotOpenError(event));
                this._sentRequestsQueue.delete(key);
            });
        }
        this._removeSocketListeners();
    }
}
exports.SocketProvider = SocketProvider;
//# sourceMappingURL=socket_provider.js.map