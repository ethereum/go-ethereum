import { ArtifactResolver } from "./types/artifact";
import { StatusResult } from "./types/status";
/**
 * Show the status of a deployment.
 *
 * @param deploymentDir - the directory of the deployment to get the status of
 * @param artifactResolver - the artifact resolver to use when loading artifacts
 * for a future
 *
 * @beta
 */
export declare function status(deploymentDir: string, artifactResolver: Omit<ArtifactResolver, "getBuildInfo">): Promise<StatusResult>;
//# sourceMappingURL=status.d.ts.map