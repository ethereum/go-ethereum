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
exports.IpcProvider = void 0;
var net_1 = require("net");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var json_rpc_provider_1 = require("./json-rpc-provider");
var IpcProvider = /** @class */ (function (_super) {
    __extends(IpcProvider, _super);
    function IpcProvider(path, network) {
        var _this = this;
        if (path == null) {
            logger.throwError("missing path", logger_1.Logger.errors.MISSING_ARGUMENT, { arg: "path" });
        }
        _this = _super.call(this, "ipc://" + path, network) || this;
        (0, properties_1.defineReadOnly)(_this, "path", path);
        return _this;
    }
    // @TODO: Create a connection to the IPC path and use filters instead of polling for block
    IpcProvider.prototype.send = function (method, params) {
        // This method is very simple right now. We create a new socket
        // connection each time, which may be slower, but the main
        // advantage we are aiming for now is security. This simplifies
        // multiplexing requests (since we do not need to multiplex).
        var _this = this;
        var payload = JSON.stringify({
            method: method,
            params: params,
            id: 42,
            jsonrpc: "2.0"
        });
        return new Promise(function (resolve, reject) {
            var response = Buffer.alloc(0);
            var stream = (0, net_1.connect)(_this.path);
            stream.on("data", function (data) {
                response = Buffer.concat([response, data]);
            });
            stream.on("end", function () {
                try {
                    resolve(JSON.parse(response.toString()).result);
                    // @TODO: Better pull apart the error
                    stream.destroy();
                }
                catch (error) {
                    reject(error);
                    stream.destroy();
                }
            });
            stream.on("error", function (error) {
                reject(error);
                stream.destroy();
            });
            stream.write(payload);
            stream.end();
        });
    };
    return IpcProvider;
}(json_rpc_provider_1.JsonRpcProvider));
exports.IpcProvider = IpcProvider;
//# sourceMappingURL=ipc-provider.js.map