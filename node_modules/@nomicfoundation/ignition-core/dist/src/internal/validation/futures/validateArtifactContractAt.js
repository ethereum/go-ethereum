"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateArtifactContractAt = void 0;
const errors_1 = require("../../../errors");
const type_guards_1 = require("../../../type-guards");
const errors_list_1 = require("../../errors-list");
async function validateArtifactContractAt(future, _artifactLoader, deploymentParameters, _accounts) {
    const errors = [];
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
exports.validateArtifactContractAt = validateArtifactContractAt;
//# sourceMappingURL=validateArtifactContractAt.js.map