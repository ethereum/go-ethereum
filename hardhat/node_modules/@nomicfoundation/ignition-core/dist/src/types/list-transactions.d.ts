import type { SolidityParameterType } from "./module";
/**
 * The status of a transaction.
 *
 * @beta
 */
export declare enum TransactionStatus {
    SUCCESS = "SUCCESS",
    FAILURE = "FAILURE",
    DROPPED = "DROPPED",
    PENDING = "PENDING"
}
/**
 * The information of a transaction.
 *
 * @beta
 */
export interface TransactionInfo {
    type: string;
    status: TransactionStatus;
    txHash: string;
    from: string;
    to?: string;
    name?: string;
    address?: string;
    params?: SolidityParameterType[];
    value?: bigint;
    browserUrl?: string;
}
/**
 * An array of transaction information.
 *
 * @beta
 */
export type ListTransactionsResult = TransactionInfo[];
//# sourceMappingURL=list-transactions.d.ts.map