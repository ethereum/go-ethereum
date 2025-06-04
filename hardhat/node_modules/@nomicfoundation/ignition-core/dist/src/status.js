"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.status = void 0;
const errors_1 = require("./errors");
const file_deployment_loader_1 = require("./internal/deployment-loader/file-deployment-loader");
const errors_list_1 = require("./internal/errors-list");
const deployment_state_helpers_1 = require("./internal/execution/deployment-state-helpers");
const find_deployed_contracts_1 = require("./internal/views/find-deployed-contracts");
const find_status_1 = require("./internal/views/find-status");
/**
 * Show the status of a deployment.
 *
 * @param deploymentDir - the directory of the deployment to get the status of
 * @param _artifactResolver - DEPRECATED: this parameter is not used and will be removed in the future
 *
 * @beta
 */
async function status(deploymentDir, _artifactResolver) {
    const deploymentLoader = new file_deployment_loader_1.FileDeploymentLoader(deploymentDir);
    const deploymentState = await (0, deployment_state_helpers_1.loadDeploymentState)(deploymentLoader);
    if (deploymentState === undefined) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.STATUS.UNINITIALIZED_DEPLOYMENT, {
            deploymentDir,
        });
    }
    const futureStatuses = (0, find_status_1.findStatus)(deploymentState);
    const deployedContracts = (0, find_deployed_contracts_1.findDeployedContracts)(deploymentState);
    const contracts = {};
    for (const [futureId, deployedContract] of Object.entries(deployedContracts)) {
        const artifact = await deploymentLoader.loadArtifact(deployedContract.id);
        contracts[futureId] = {
            ...deployedContract,
            contractName: artifact.contractName,
            sourceName: artifact.sourceName,
            abi: artifact.abi,
        };
    }
    const statusResult = {
        ...futureStatuses,
        chainId: deploymentState.chainId,
        contracts,
    };
    return statusResult;
}
exports.status = status;
//# sourceMappingURL=status.js.map