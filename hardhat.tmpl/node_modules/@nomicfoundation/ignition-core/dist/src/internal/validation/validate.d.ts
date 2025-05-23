import { ArtifactResolver } from "../../types/artifact";
import { DeploymentParameters, ValidationErrorDeploymentResult } from "../../types/deploy";
import { IgnitionModule } from "../../types/module";
export declare function validate(module: IgnitionModule, artifactLoader: ArtifactResolver, deploymentParameters: DeploymentParameters, accounts: string[]): Promise<ValidationErrorDeploymentResult | null>;
//# sourceMappingURL=validate.d.ts.map