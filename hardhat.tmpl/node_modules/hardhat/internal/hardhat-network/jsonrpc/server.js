"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.JsonRpcServer = void 0;
const debug_1 = __importDefault(require("debug"));
const http_1 = __importDefault(require("http"));
const http_2 = require("../../core/providers/http");
const handler_1 = require("./handler");
const log = (0, debug_1.default)("hardhat:core:hardhat-network:jsonrpc");
class JsonRpcServer {
    constructor(config) {
        this.getProvider = (name = "json-rpc") => {
            const { Client } = require("undici");
            const { address, port } = this._httpServer.address();
            const dispatcher = new Client(`http://${address}:${port}/`, {
                keepAliveTimeout: 10,
                keepAliveMaxTimeout: 10,
            });
            return new http_2.HttpProvider(`http://${address}:${port}/`, name, {}, 20000, dispatcher);
        };
        this.listen = () => {
            return new Promise((resolve) => {
                log(`Starting JSON-RPC server on port ${this._config.port}`);
                this._httpServer.listen(this._config.port, this._config.hostname, () => {
                    // We get the address and port directly from the server in order to handle random port allocation with `0`.
                    const address = this._httpServer.address(); // TCP sockets return AddressInfo
                    resolve(address);
                });
            });
        };
        this.waitUntilClosed = async () => {
            const httpServerClosed = new Promise((resolve) => {
                this._httpServer.once("close", resolve);
            });
            const wsServerClosed = new Promise((resolve) => {
                this._wsServer.once("close", resolve);
            });
            await Promise.all([httpServerClosed, wsServerClosed]);
        };
        this.close = async () => {
            await Promise.all([
                new Promise((resolve, reject) => {
                    log("Closing JSON-RPC server");
                    this._httpServer.close((err) => {
                        if (err !== null && err !== undefined) {
                            log("Failed to close JSON-RPC server");
                            reject(err);
                            return;
                        }
                        log("JSON-RPC server closed");
                        resolve();
                    });
                }),
                new Promise((resolve, reject) => {
                    log("Closing websocket server");
                    this._wsServer.close((err) => {
                        if (err !== null && err !== undefined) {
                            log("Failed to close websocket server");
                            reject(err);
                            return;
                        }
                        log("Websocket server closed");
                        resolve();
                    });
                }),
            ]);
        };
        const { Server: WSServer } = require("ws");
        this._config = config;
        const handler = new handler_1.JsonRpcHandler(config.provider);
        this._httpServer = http_1.default.createServer();
        this._wsServer = new WSServer({
            server: this._httpServer,
        });
        this._httpServer.on("request", handler.handleHttp);
        this._wsServer.on("connection", handler.handleWs);
    }
}
exports.JsonRpcServer = JsonRpcServer;
//# sourceMappingURL=server.js.map