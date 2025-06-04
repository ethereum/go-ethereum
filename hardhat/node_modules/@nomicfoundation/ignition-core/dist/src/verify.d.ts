import { BuildInfo } from "./types/artifact";
import { ChainConfig, VerifyResult } from "./types/verify";
/**
 * Retrieve the information required to verify all contracts from a deployment on Etherscan.
 *
 * @param deploymentDir - the file directory of the deployment
 * @param customChains - an array of custom chain configurations
 *
 * @beta
 */
export declare function getVerificationInformation(deploymentDir: string, customChains?: ChainConfig[], includeUnrelatedContracts?: boolean): AsyncGenerator<VerifyResult>;
export declare function getImportSourceNames(sourceName: string, buildInfo: BuildInfo, visited?: Record<string, boolean>): string[];
//# sourceMappingURL=verify.d.ts.map