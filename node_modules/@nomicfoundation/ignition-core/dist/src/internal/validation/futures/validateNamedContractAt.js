"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateNamedContractAt = void 0;
const errors_1 = require("../../../errors");
const type_guards_1 = require("../../../type-guards");
const errors_list_1 = require("../../errors-list");
async function validateNamedContractAt(future, artifactLoader, deploymentParameters, _accounts) {
    const errors = [];
    /* stage one */
    const artifact = await artifactLoader.loadArtifact(future.contractName);
    if (!(0, type_guards_1.isArtifactType)(artifact)) {
        errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.INVALID_ARTIFACT, {
            contractName: future.contractName,
        }));
    }
    /* stage two */
    if ((0, type_guards_1.isModuleParameterRuntimeValue)(future.address)) {
        const param = deploymentParameters[future.address.moduleId]?.[future.address.name] ??
            future.address.defaultValue;
        if (param === undefined) {
            errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.MISSING_MODULE_PARAMETER, {
                name: future.address.name,
            }));
        }
        else if (typeof param !== "string") {
            errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.INVALID_MODULE_PARAMETER_TYPE, {
                name: future.address.name,
                expectedType: "string",
                actualType: typeof param,
            }));
        }
    }
    return errors.map((e) => e.message);
}
exports.validateNamedContractAt = validateNamedContractAt;
//# sourceMappingURL=validateNamedContractAt.js.map