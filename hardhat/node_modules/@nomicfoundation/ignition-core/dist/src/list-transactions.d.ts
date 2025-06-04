import type { ArtifactResolver } from "./types/artifact";
import { type ListTransactionsResult } from "./types/list-transactions";
/**
 * Returns the transactions associated with a deployment.
 *
 * @param deploymentDir - the directory of the deployment to get the transactions of
 * @param artifactResolver - the artifact resolver to use when loading artifacts
 * for a future
 *
 * @beta
 */
export declare function listTransactions(deploymentDir: string, _artifactResolver: Omit<ArtifactResolver, "getBuildInfo">): Promise<ListTransactionsResult>;
//# sourceMappingURL=list-transactions.d.ts.map