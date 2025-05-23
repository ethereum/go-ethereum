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
Object.defineProperty(exports, "__esModule", { value: true });
exports.IpcProvider = void 0;
const net_1 = require("net");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const fs_1 = require("fs");
/**
 * The IPC Provider could be used in node.js dapps when running a local node. And it provide the most secure connection.
 *
 * @example
 * ```ts
 * const provider = new IpcProvider(
 * 		`path.ipc`,
 * 		{
 * 			writable: false,
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
 * const provider = new IpcProvider(
 * 		`path.ipc`,
 * 		{},
 * 		{
 * 			delay: 500,
 * 			autoReconnect: true,
 * 			maxAttempts: 10,
 * 		},
 * 	);
 * ```
 */
class IpcProvider extends web3_utils_1.SocketProvider {
    /**
     * This is a class used for IPC connections. It extends the abstract class SocketProvider {@link SocketProvider} that extends the EIP-1193 provider {@link EIP1193Provider}.
     * @param socketPath - The path to the IPC socket.
     * @param socketOptions - The options for the IPC socket connection.
     * @param reconnectOptions - The options for the socket reconnection {@link ReconnectOptions}
     */
    // this constructor is to specify the type for `socketOptions` for a better intellisense.
    // eslint-disable-next-line no-useless-constructor
    constructor(socketPath, socketOptions, reconnectOptions) {
        super(socketPath, socketOptions, reconnectOptions);
    }
    getStatus() {
        var _a;
        if ((_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.connecting) {
            return 'connecting';
        }
        return this._connectionStatus;
    }
    _openSocketConnection() {
        if (!(0, fs_1.existsSync)(this._socketPath)) {
            throw new web3_errors_1.InvalidClientError(this._socketPath);
        }
        if (!this._socketConnection || this.getStatus() === 'disconnected') {
            this._socketConnection = new net_1.Socket(this._socketOptions);
        }
        this._socketConnection.connect({ path: this._socketPath });
    }
    _closeSocketConnection(code, data) {
        var _a;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.end(() => {
            this._onDisconnect(code, data);
        });
    }
    _sendToSocket(payload) {
        var _a;
        if (this.getStatus() === 'disconnected') {
            throw new web3_errors_1.ConnectionNotOpenError();
        }
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.write(JSON.stringify(payload));
    }
    _parseResponses(e) {
        return this.chunkResponseParser.parseResponse(typeof e === 'string' ? e : (0, web3_utils_1.toUtf8)(e));
    }
    _addSocketListeners() {
        var _a, _b, _c, _d, _e;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.on('data', this._onMessageHandler);
        (_b = this._socketConnection) === null || _b === void 0 ? void 0 : _b.on('connect', this._onOpenHandler);
        (_c = this._socketConnection) === null || _c === void 0 ? void 0 : _c.on('close', this._onClose.bind(this));
        (_d = this._socketConnection) === null || _d === void 0 ? void 0 : _d.on('end', this._onCloseHandler);
        (_e = this._socketConnection) === null || _e === void 0 ? void 0 : _e.on('error', this._onErrorHandler);
    }
    _removeSocketListeners() {
        var _a, _b, _c, _d;
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.removeAllListeners('connect');
        (_b = this._socketConnection) === null || _b === void 0 ? void 0 : _b.removeAllListeners('end');
        (_c = this._socketConnection) === null || _c === void 0 ? void 0 : _c.removeAllListeners('close');
        (_d = this._socketConnection) === null || _d === void 0 ? void 0 : _d.removeAllListeners('data');
        // note: we intentionally keep the error event listener to be able to emit it in case an error happens when closing the connection
    }
    _onCloseEvent(event) {
        var _a;
        if (!event && this._reconnectOptions.autoReconnect) {
            this._connectionStatus = 'disconnected';
            this._reconnect();
            return;
        }
        this._clearQueues(event);
        this._removeSocketListeners();
        this._onDisconnect(event === null || event === void 0 ? void 0 : event.code, event === null || event === void 0 ? void 0 : event.reason);
        // disconnect was successful and can safely remove error listener
        (_a = this._socketConnection) === null || _a === void 0 ? void 0 : _a.removeAllListeners('error');
    }
    _onClose(event) {
        this._clearQueues(event);
        this._removeSocketListeners();
    }
}
exports.default = IpcProvider;
exports.IpcProvider = IpcProvider;
//# sourceMappingURL=index.js.map