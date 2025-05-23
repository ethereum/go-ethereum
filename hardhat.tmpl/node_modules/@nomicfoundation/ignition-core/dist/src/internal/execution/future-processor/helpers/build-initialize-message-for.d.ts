import { DeploymentParameters } from "../../../../types/deploy";
import { Future } from "../../../../types/module";
import { DeploymentLoader } from "../../../deployment-loader/types";
import { DeploymentState } from "../../types/deployment-state";
import { ExecutionStrategy } from "../../types/execution-strategy";
import { JournalMessage } from "../../types/messages";
export declare function buildInitializeMessageFor(future: Future, deploymentState: DeploymentState, strategy: ExecutionStrategy, deploymentParameters: DeploymentParameters, deploymentLoader: DeploymentLoader, accounts: string[], defaultSender: string): Promise<JournalMessage>;
//# sourceMappingURL=build-initialize-message-for.d.ts.map