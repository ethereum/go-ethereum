"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.wipe = void 0;
const ephemeral_deployment_loader_1 = require("./internal/deployment-loader/ephemeral-deployment-loader");
const file_deployment_loader_1 = require("./internal/deployment-loader/file-deployment-loader");
const wiper_1 = require("./internal/wiper");
/**
 * Clear the state against a future within a deployment
 *
 * @param deploymentDir - the file directory of the deployment
 * @param futureId - the future to be cleared
 *
 * @beta
 */
async function wipe(deploymentDir, artifactResolver, futureId) {
    const deploymentLoader = deploymentDir !== undefined
        ? new file_deployment_loader_1.FileDeploymentLoader(deploymentDir)
        : new ephemeral_deployment_loader_1.EphemeralDeploymentLoader(artifactResolver);
    const wiper = new wiper_1.Wiper(deploymentLoader);
    await wiper.wipe(futureId);
}
exports.wipe = wipe;
//# sourceMappingURL=wipe.js.map