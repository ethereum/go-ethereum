"use strict";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
let WS = null;
try {
    WS = WebSocket;
    if (WS == null) {
        throw new Error("inject please");
    }
}
catch (error) {
    const logger = new Logger(version);
    WS = function () {
        logger.throwError("WebSockets not supported in this environment", Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "new WebSocket()"
        });
    };
}
//export default WS;
//module.exports = WS;
export { WS as WebSocket };
//# sourceMappingURL=ws.js.map