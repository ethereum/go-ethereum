/// <reference types="node" />
import { JsonRpcProvider } from "./json-rpc-provider";
export declare class JsonRpcBatchProvider extends JsonRpcProvider {
    _pendingBatchAggregator: NodeJS.Timer;
    _pendingBatch: Array<{
        request: {
            method: string;
            params: Array<any>;
            id: number;
            jsonrpc: "2.0";
        };
        resolve: (result: any) => void;
        reject: (error: Error) => void;
    }>;
    send(method: string, params: Array<any>): Promise<any>;
}
//# sourceMappingURL=json-rpc-batch-provider.d.ts.map