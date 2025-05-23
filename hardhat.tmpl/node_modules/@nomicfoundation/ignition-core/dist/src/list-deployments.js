"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.listDeployments = void 0;
const fs_extra_1 = require("fs-extra");
/**
 * Return a list of all deployments in the deployment directory.
 *
 * @param deploymentDir - the directory of the deployments
 *
 * @beta
 */
async function listDeployments(deploymentDir) {
    if (!(await (0, fs_extra_1.pathExists)(deploymentDir))) {
        return [];
    }
    return (0, fs_extra_1.readdir)(deploymentDir);
}
exports.listDeployments = listDeployments;
//# sourceMappingURL=list-deployments.js.map