import { ArtifactResolver } from "../../types/artifact";
import { DeploymentParameters } from "../../types/deploy";
import { IgnitionModule } from "../../types/module";
import { DeploymentLoader } from "../deployment-loader/types";
import { DeploymentState } from "../execution/types/deployment-state";
import { ConcreteExecutionConfig } from "../execution/types/execution-state";
import { ReconciliationFailure, ReconciliationResult } from "./types";
export declare class Reconciler {
    static reconcile(module: IgnitionModule, deploymentState: DeploymentState, deploymentParameters: DeploymentParameters, accounts: string[], deploymentLoader: DeploymentLoader, artifactResolver: ArtifactResolver, defaultSender: string, strategy: string, strategyConfig: ConcreteExecutionConfig): Promise<ReconciliationResult>;
    static checkForPreviousRunErrors(deploymentState: DeploymentState): ReconciliationFailure[];
    private static _previousRunFailedMessageFor;
    private static _reconcileEachFutureInModule;
    private static _missingPreviouslyExecutedFutures;
    private static _getFuturesInReverseTopoligicalOrder;
    private static _check;
}
//# sourceMappingURL=reconciler.d.ts.map