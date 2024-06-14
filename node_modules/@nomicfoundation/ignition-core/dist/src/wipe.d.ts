import { ArtifactResolver } from "./types/artifact";
/**
 * Clear the state against a future within a deployment
 *
 * @param deploymentDir - the file directory of the deployment
 * @param futureId - the future to be cleared
 *
 * @beta
 */
export declare function wipe(deploymentDir: string, artifactResolver: ArtifactResolver, futureId: string): Promise<void>;
//# sourceMappingURL=wipe.d.ts.map