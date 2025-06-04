"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.saveArtifactsForFuture = void 0;
const module_1 = require("../../../../types/module");
async function saveArtifactsForFuture(future, artifactResolver, deploymentLoader) {
    switch (future.type) {
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
        case module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT:
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
            return _storeArtifactAndBuildInfoAgainstDeployment(future, {
                artifactResolver,
                deploymentLoader,
            });
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
        case module_1.FutureType.LIBRARY_DEPLOYMENT:
        case module_1.FutureType.CONTRACT_AT:
            return deploymentLoader.storeUserProvidedArtifact(future.id, future.artifact);
        case module_1.FutureType.CONTRACT_CALL:
        case module_1.FutureType.STATIC_CALL:
        case module_1.FutureType.ENCODE_FUNCTION_CALL:
        case module_1.FutureType.READ_EVENT_ARGUMENT:
        case module_1.FutureType.SEND_DATA:
            return;
    }
}
exports.saveArtifactsForFuture = saveArtifactsForFuture;
async function _storeArtifactAndBuildInfoAgainstDeployment(future, { deploymentLoader, artifactResolver, }) {
    const artifact = await artifactResolver.loadArtifact(future.contractName);
    await deploymentLoader.storeNamedArtifact(future.id, future.contractName, artifact);
    const buildInfo = await artifactResolver.getBuildInfo(future.contractName);
    if (buildInfo !== undefined) {
        await deploymentLoader.storeBuildInfo(future.id, buildInfo);
    }
}
//# sourceMappingURL=save-artifacts-for-future.js.map