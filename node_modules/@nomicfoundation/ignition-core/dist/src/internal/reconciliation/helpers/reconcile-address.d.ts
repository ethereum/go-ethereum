import { ContractAtFuture, NamedArtifactContractAtFuture } from "../../../types/module";
import { ContractAtExecutionState } from "../../execution/types/execution-state";
import { ReconciliationContext, ReconciliationFutureResultFailure } from "../types";
export declare function reconcileAddress(future: NamedArtifactContractAtFuture<string> | ContractAtFuture, exState: ContractAtExecutionState, context: ReconciliationContext): ReconciliationFutureResultFailure | undefined;
//# sourceMappingURL=reconcile-address.d.ts.map