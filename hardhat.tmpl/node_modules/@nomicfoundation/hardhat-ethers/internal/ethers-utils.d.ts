import type { TransactionRequest, PreparedTransactionRequest, BlockParams, TransactionResponseParams, TransactionReceiptParams, LogParams, JsonRpcTransactionRequest } from "ethers";
export type FormatFunc = (value: any) => any;
export declare function copyRequest(req: TransactionRequest): PreparedTransactionRequest;
export declare function resolveProperties<T>(value: {
    [P in keyof T]: T[P] | Promise<T[P]>;
}): Promise<T>;
export declare function formatBlock(value: any): BlockParams;
export declare function formatTransactionResponse(value: any): TransactionResponseParams;
export declare function formatTransactionReceipt(value: any): TransactionReceiptParams;
export declare function formatReceiptLog(value: any): LogParams;
export declare function formatLog(value: any): LogParams;
export declare function getRpcTransaction(tx: TransactionRequest): JsonRpcTransactionRequest;
//# sourceMappingURL=ethers-utils.d.ts.map