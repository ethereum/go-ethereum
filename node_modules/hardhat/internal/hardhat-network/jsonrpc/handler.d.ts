import { IncomingMessage, ServerResponse } from "http";
import WebSocket from "ws";
import { EIP1193Provider } from "../../../types";
export declare class JsonRpcHandler {
    private readonly _provider;
    constructor(_provider: EIP1193Provider);
    handleHttp: (req: IncomingMessage, res: ServerResponse) => Promise<void>;
    handleWs: (ws: WebSocket) => Promise<void>;
    private _sendEmptyResponse;
    private _setCorsHeaders;
    private _sendResponse;
    private _handleSingleRequest;
    private _handleSingleWsRequest;
    private _handleRequest;
}
//# sourceMappingURL=handler.d.ts.map