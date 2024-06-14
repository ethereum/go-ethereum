"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WebSocket = void 0;
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var WS = null;
exports.WebSocket = WS;
try {
    exports.WebSocket = WS = WebSocket;
    if (WS == null) {
        throw new Error("inject please");
    }
}
catch (error) {
    var logger_2 = new logger_1.Logger(_version_1.version);
    exports.WebSocket = WS = function () {
        logger_2.throwError("WebSockets not supported in this environment", logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "new WebSocket()"
        });
    };
}
//# sourceMappingURL=browser-ws.js.map