"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validate = void 0;
const deploy_1 = require("../../types/deploy");
const module_1 = require("../../types/module");
const get_futures_from_module_1 = require("../utils/get-futures-from-module");
const validateArtifactContractAt_1 = require("./futures/validateArtifactContractAt");
const validateArtifactContractDeployment_1 = require("./futures/validateArtifactContractDeployment");
const validateArtifactLibraryDeployment_1 = require("./futures/validateArtifactLibraryDeployment");
const validateNamedContractAt_1 = require("./futures/validateNamedContractAt");
const validateNamedContractCall_1 = require("./futures/validateNamedContractCall");
const validateNamedContractDeployment_1 = require("./futures/validateNamedContractDeployment");
const validateNamedEncodeFunctionCall_1 = require("./futures/validateNamedEncodeFunctionCall");
const validateNamedLibraryDeployment_1 = require("./futures/validateNamedLibraryDeployment");
const validateNamedStaticCall_1 = require("./futures/validateNamedStaticCall");
const validateReadEventArgument_1 = require("./futures/validateReadEventArgument");
const validateSendData_1 = require("./futures/validateSendData");
async function validate(module, artifactLoader, deploymentParameters, accounts) {
    const futures = (0, get_futures_from_module_1.getFuturesFromModule)(module);
    const errors = {};
    for (const future of futures) {
        const validationErrors = await _validateFuture(future, artifactLoader, deploymentParameters, accounts);
        if (validationErrors.length > 0) {
            errors[future.id] = validationErrors;
        }
    }
    if (Object.keys(errors).length === 0) {
        // No validation errors
        return null;
    }
    return {
        type: deploy_1.DeploymentResultType.VALIDATION_ERROR,
        errors,
    };
}
exports.validate = validate;
async function _validateFuture(future, artifactLoader, deploymentParameters, accounts) {
    switch (future.type) {
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
            return (0, validateArtifactContractDeployment_1.validateArtifactContractDeployment)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.LIBRARY_DEPLOYMENT:
            return (0, validateArtifactLibraryDeployment_1.validateArtifactLibraryDeployment)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.CONTRACT_AT:
            return (0, validateArtifactContractAt_1.validateArtifactContractAt)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
            return (0, validateNamedContractDeployment_1.validateNamedContractDeployment)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT:
            return (0, validateNamedLibraryDeployment_1.validateNamedLibraryDeployment)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
            return (0, validateNamedContractAt_1.validateNamedContractAt)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.CONTRACT_CALL:
            return (0, validateNamedContractCall_1.validateNamedContractCall)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.STATIC_CALL:
            return (0, validateNamedStaticCall_1.validateNamedStaticCall)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.ENCODE_FUNCTION_CALL:
            return (0, validateNamedEncodeFunctionCall_1.validateNamedEncodeFunctionCall)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.READ_EVENT_ARGUMENT:
            return (0, validateReadEventArgument_1.validateReadEventArgument)(future, artifactLoader, deploymentParameters, accounts);
        case module_1.FutureType.SEND_DATA:
            return (0, validateSendData_1.validateSendData)(future, artifactLoader, deploymentParameters, accounts);
    }
}
//# sourceMappingURL=validate.js.map