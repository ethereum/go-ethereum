import { ContractCallFuture, ContractDeploymentFuture, NamedArtifactContractDeploymentFuture, SendDataFuture } from "../../../types/module";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState } from "../../execution/types/execution-state";
import { ReconciliationContext, ReconciliationFutureResultFailure } from "../types";
export declare function reconcileValue(future: NamedArtifactContractDeploymentFuture<string> | ContractDeploymentFuture | ContractCallFuture<string, string> | SendDataFuture, exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState, context: ReconciliationContext): ReconciliationFutureResultFailure | undefined;
//# sourceMappingURL=reconcile-value.d.ts.map