import { ContractDeploymentFuture, LibraryDeploymentFuture, NamedArtifactContractDeploymentFuture, NamedArtifactLibraryDeploymentFuture } from "../../../types/module";
import { DeploymentExecutionState } from "../../execution/types/execution-state";
import { ReconciliationContext, ReconciliationFutureResultFailure } from "../types";
export declare function reconcileLibraries(future: NamedArtifactContractDeploymentFuture<string> | ContractDeploymentFuture | NamedArtifactLibraryDeploymentFuture<string> | LibraryDeploymentFuture, exState: DeploymentExecutionState, context: ReconciliationContext): ReconciliationFutureResultFailure | undefined;
//# sourceMappingURL=reconcile-libraries.d.ts.map