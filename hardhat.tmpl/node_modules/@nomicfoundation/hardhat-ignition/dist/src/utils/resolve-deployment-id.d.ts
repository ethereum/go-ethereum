/**
 * Determine the deploymentId, using either the user provided id,
 * throwing if it is invalid, or generating one from the chainId
 * if none was provided.
 *
 * @param givenDeploymentId - the user provided deploymentId if
 * they provided one undefined otherwise
 * @param chainId - the chainId of the network being deployed to
 *
 * @returns the deploymentId
 */
export declare function resolveDeploymentId(givenDeploymentId: string | undefined, chainId: number): string;
/**
 * Determine if the given identifier the rules for a valid deployment id.
 * */
export declare function _isValidDeploymentIdentifier(identifier: string): boolean;
//# sourceMappingURL=resolve-deployment-id.d.ts.map