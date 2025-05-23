import { IgnitionModule, IgnitionModuleResult } from "./types/module";
/**
 * Provides a array of batches, where each batch is an array of futureIds,
 * based on Ignition's batching algorithm, assuming a the module is being
 * run from as a fresh deployment.
 *
 * @param ignitionModule - the Ignition module to be get batch information for
 * @returns the batches Ignition will use for the module
 *
 * @beta
 */
export declare function batches(ignitionModule: IgnitionModule<string, string, IgnitionModuleResult<string>>): string[][];
//# sourceMappingURL=batches.d.ts.map