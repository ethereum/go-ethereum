import { DeploymentState } from "../execution/types/deployment-state";
import { Transaction, TransactionReceipt } from "../execution/types/jsonrpc";
export declare function findConfirmedTransactionByFutureId(deploymentState: DeploymentState, futureId: string): Omit<Transaction, "receipt"> & {
    receipt: TransactionReceipt;
};
//# sourceMappingURL=find-confirmed-transaction-by-future-id.d.ts.map