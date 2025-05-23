"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileArtifacts = void 0;
const utils_1 = require("../utils");
async function reconcileArtifacts(future, exState, context) {
    const moduleArtifact = "artifact" in future
        ? future.artifact
        : await context.artifactResolver.loadArtifact(future.contractName);
    const storedArtifact = await context.deploymentLoader.loadArtifact(exState.artifactId);
    const moduleArtifactBytecode = (0, utils_1.getBytecodeWithoutMetadata)(moduleArtifact.bytecode);
    const storedArtifactBytecode = (0, utils_1.getBytecodeWithoutMetadata)(storedArtifact.bytecode);
    if (moduleArtifactBytecode !== storedArtifactBytecode) {
        return (0, utils_1.fail)(future, "Artifact bytecodes have been changed");
    }
}
exports.reconcileArtifacts = reconcileArtifacts;
//# sourceMappingURL=reconcile-artifacts.js.map