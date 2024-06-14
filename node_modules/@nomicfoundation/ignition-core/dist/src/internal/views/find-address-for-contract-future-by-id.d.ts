import { DeploymentState } from "../execution/types/deployment-state";
/**
 * Find the address for the future by its id. Only works for ContractAt, NamedLibrary,
 * NamedContract, ArtifactLibrary, ArtifactContract as only they result in an
 * address on completion.
 *
 * Assumes that the future has been completed.
 *
 * @param deploymentState
 * @param futureId
 * @returns
 */
export declare function findAddressForContractFuture(deploymentState: DeploymentState, futureId: string): string;
//# sourceMappingURL=find-address-for-contract-future-by-id.d.ts.map