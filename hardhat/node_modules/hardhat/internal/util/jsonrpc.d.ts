export interface JsonRpcRequest {
    jsonrpc: string;
    method: string;
    params: any[];
    id: number | string;
}
export interface SuccessfulJsonRpcResponse {
    jsonrpc: string;
    id: number | string;
    result: any;
}
export interface FailedJsonRpcResponse {
    jsonrpc: string;
    id: number | string | null;
    error: {
        code: number;
        message: string;
        data?: any;
    };
}
export type JsonRpcResponse = SuccessfulJsonRpcResponse | FailedJsonRpcResponse;
export declare function parseJsonResponse(text: string): JsonRpcResponse | JsonRpcResponse[];
export declare function isValidJsonRequest(payload: any): boolean;
export declare function isValidJsonResponse(payload: any): boolean;
export declare function isSuccessfulJsonResponse(payload: JsonRpcResponse): payload is SuccessfulJsonRpcResponse;
//# sourceMappingURL=jsonrpc.d.ts.map