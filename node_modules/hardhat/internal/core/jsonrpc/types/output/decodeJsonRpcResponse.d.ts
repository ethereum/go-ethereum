import * as t from "io-ts";
/**
 * This function decodes an RPC out type, throwing InvalidResponseError if it's not valid.
 */
export declare function decodeJsonRpcResponse<T>(value: unknown, codec: t.Type<T>): T;
//# sourceMappingURL=decodeJsonRpcResponse.d.ts.map