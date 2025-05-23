"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.WebSocketProvider = void 0;
const isomorphic_ws_1 = __importDefault(require("isomorphic-ws"));
const web3_utils_1 = require("web3-utils");
const web3_errors_1 = require("web3-errors");
/**
 * Use WebSocketProvider to connect to a Node using a WebSocket connection, i.e. over the `ws` or `wss` protocol.
 *
 * @example
 * ```ts
 * const provider = new WebSocketProvider(
 * 		`ws://localhost:8545`,
 * 		{
 * 			headers: {
 * 				// to provide the API key if the Node requires the key to be inside the `headers` for example:
 * 				'x-api-key': '<Api key>',
 * 			},
 * 		},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 *
 * The second and the third parameters are both optional. And you can for example, the second parameter could be an empty object or undefined.
 *  * @example
 * ```ts
 * const provider = new WebSocketProvider(
 * 		`ws://localhost:8545`,
 * 		{},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 */
class WebSocketProvider extends web3_utils_1.SocketProvider {
    /**
     * This is a class used for Web Socket connections. It extends the abstract class SocketProvider {@link SocketProvider} that extends the EIP-1193 provider {@link EIP1193Provider}.
     * @param socketPath - The path to the Web Socket.
     * @param socketOptions - The options for the Web Socket client.
     * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
     */
    // this constructor is to specify the type for `socketOptions` for a better intellisense.
    // eslint-disable-next-line no-useless-constructor
    constructor(socketPath, socketOptions, reconnectOptions) {
        super(socketPath, socketOptions, reconnectOptions);
    }
    // eslint-disable-next-line class-methods-use-this
    _validateProviderPath(providerUrl) {
        return typeof providerUrl === 'string' ? /^ws(s)?:\/\//i.test(providerUrl) : false;
    }
    getStatus() {
        if (this._socketConnection && !(0, web3_utils_1.isNullish)(this._socketConnection)) {
            switch (this._socketConnection.readyState) {
                case this._socketConnection.CONNECTING: {
                    return 'connecting';
                }
                case this._socketConnection.OPEN: {
                    return 'connected';
                }
                default: {
                    return 'disconnected';
                }
            }
        }
        return 'disconnected';
    }
    _openSocketConnection() {
        this._socketConnection = new isomorphic_ws_1.default(this._socketPath, undefined, this._socketOptions && Object.keys(this._socketOptions).length === 0
            ? undefined
            : this._socketOptions);
    }
    _closeSocketConnection(code, data) {
        var _a;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.close(code, data);
    }
    _sendToSocket(payload) {
        var _a;
        if (this.getStatus() === 'disconnected') {
            throw new web3_errors_1.ConnectionNotOpenError();
        }
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.send(JSON.stringify(payload));
    }
    _parseResponses(event) {
        return this.chunkResponseParser.parseResponse(event.data);
    }
    _addSocketListeners() {
        var _a, _b, _c, _d;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.addEventListener('open', this._onOpenHandler);
        (_b = this._socketConnection) === null || _b === void 0 ? void 0 : _b.addEventListener('message', this._onMessageHandler);
        (_c = this._socketConnection) === null || _c === void 0 ? void 0 : _c.addEventListener('close', e => this._onCloseHandler(e));
        (_d = this._socketConnection) === null || _d === void 0 ? void 0 : _d.addEventListener('error', this._onErrorHandler);
    }
    _removeSocketListeners() {
        var _a, _b, _c;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.removeEventListener('message', this._onMessageHandler);
        (_b = this._socketConnection) === null || _b === void 0 ? void 0 : _b.removeEventListener('open', this._onOpenHandler);
        (_c = this._socketConnection) === null || _c === void 0 ? void 0 : _c.removeEventListener('close', this._onCloseHandler);
        // note: we intentionally keep the error event listener to be able to emit it in case an error happens when closing the connection
    }
    _onCloseEvent(event) {
        var _a;
        if (this._reconnectOptions.autoReconnect &&
            (![1000, 1001].includes(event.code) || !event.wasClean)) {
            this._reconnect();
            return;
        }
        this._clearQueues(event);
        this._removeSocketListeners();
        this._onDisconnect(event.code, event.reason);
        // disconnect was successful and can safely remove error listener
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.removeEventListener('error', this._onErrorHandler);
    }
}
exports.default = WebSocketProvider;
exports.WebSocketProvider = WebSocketProvider;
//# sourceMappingURL=index.js.map