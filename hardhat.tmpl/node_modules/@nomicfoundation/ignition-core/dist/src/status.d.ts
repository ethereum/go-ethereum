import { ArtifactResolver } from "./types/artifact";
import { StatusResult } from "./types/status";
/**
 * Show the status of a deployment.
 *
 * @param deploymentDir - the directory of the deployment to get the status of
 * @param _artifactResolver - DEPRECATED: this parameter is not used and will be removed in the future
 *
 * @beta
 */
export declare function status(deploymentDir: string, _artifactResolver?: Omit<ArtifactResolver, "getBuildInfo">): Promise<StatusResult>;
//# sourceMappingURL=status.d.ts.map