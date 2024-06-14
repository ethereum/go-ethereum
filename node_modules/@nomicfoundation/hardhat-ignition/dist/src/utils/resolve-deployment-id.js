"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports._isValidDeploymentIdentifier = exports.resolveDeploymentId = void 0;
const plugins_1 = require("hardhat/plugins");
/**
 * A regex that captures Ignitions rules for deployment-ids, specifically
 * that they can only contain alphanumerics, dashes and underscores,
 * and that they start with a letter.
 */
const ignitionDeploymentIdRegex = /^[a-zA-Z][a-zA-Z0-9_\-]*$/;
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
function resolveDeploymentId(givenDeploymentId, chainId) {
    if (givenDeploymentId !== undefined &&
        !_isValidDeploymentIdentifier(givenDeploymentId)) {
        throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", `The deployment-id "${givenDeploymentId}" contains banned characters, ids can only contain alphanumerics, dashes or underscores`);
    }
    return givenDeploymentId ?? `chain-${chainId}`;
}
exports.resolveDeploymentId = resolveDeploymentId;
/**
 * Determine if the given identifier the rules for a valid deployment id.
 * */
function _isValidDeploymentIdentifier(identifier) {
    return ignitionDeploymentIdRegex.test(identifier);
}
exports._isValidDeploymentIdentifier = _isValidDeploymentIdentifier;
//# sourceMappingURL=resolve-deployment-id.js.map