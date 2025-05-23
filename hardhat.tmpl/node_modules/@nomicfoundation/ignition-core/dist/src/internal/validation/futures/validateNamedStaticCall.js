"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateNamedStaticCall = void 0;
const errors_1 = require("../../../errors");
const type_guards_1 = require("../../../type-guards");
const errors_list_1 = require("../../errors-list");
const abi_1 = require("../../execution/abi");
const utils_1 = require("../utils");
async function validateNamedStaticCall(future, artifactLoader, deploymentParameters, accounts) {
    const errors = [];
    /* stage one */
    const artifact = "artifact" in future.contract
        ? future.contract.artifact
        : await artifactLoader.loadArtifact(future.contract.contractName);
    if (!(0, type_guards_1.isArtifactType)(artifact)) {
        errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.INVALID_ARTIFACT, {
            contractName: future.contract.contractName,
        }));
    }
    else {
        errors.push(...(0, abi_1.validateArtifactFunction)(artifact, future.contract.contractName, future.functionName, future.args, true));
        errors.push(...(0, abi_1.validateFunctionArgumentParamType)(future.contract.contractName, future.functionName, artifact, future.nameOrIndex));
    }
    /* stage two */
    const runtimeValues = (0, utils_1.retrieveNestedRuntimeValues)(future.args);
    const moduleParams = runtimeValues.filter(type_guards_1.isModuleParameterRuntimeValue);
    const accountParams = [
        ...(0, utils_1.filterToAccountRuntimeValues)(runtimeValues),
        ...((0, type_guards_1.isAccountRuntimeValue)(future.from) ? [future.from] : []),
    ];
    errors.push(...accountParams.flatMap((arv) => (0, utils_1.validateAccountRuntimeValue)(arv, accounts)));
    const missingParams = moduleParams.filter((param) => (0, utils_1.resolvePotentialModuleParameterValueFrom)(deploymentParameters, param) ===
        undefined);
    if (missingParams.length > 0) {
        errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.MISSING_MODULE_PARAMETER, {
            name: missingParams[0].name,
        }));
    }
    return errors.map((e) => e.message);
}
exports.validateNamedStaticCall = validateNamedStaticCall;
//# sourceMappingURL=validateNamedStaticCall.js.map