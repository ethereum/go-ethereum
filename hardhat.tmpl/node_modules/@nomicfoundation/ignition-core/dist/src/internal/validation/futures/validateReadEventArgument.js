"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateReadEventArgument = void 0;
const errors_1 = require("../../../errors");
const type_guards_1 = require("../../../type-guards");
const errors_list_1 = require("../../errors-list");
const abi_1 = require("../../execution/abi");
async function validateReadEventArgument(future, artifactLoader, _deploymentParameters, _accounts) {
    const errors = [];
    /* stage one */
    const artifact = "artifact" in future.emitter
        ? future.emitter.artifact
        : await artifactLoader.loadArtifact(future.emitter.contractName);
    if (!(0, type_guards_1.isArtifactType)(artifact)) {
        errors.push(new errors_1.IgnitionError(errors_list_1.ERRORS.VALIDATION.INVALID_ARTIFACT, {
            contractName: future.emitter.contractName,
        }));
    }
    else {
        errors.push(...(0, abi_1.validateArtifactEventArgumentParams)(artifact, future.eventName, future.nameOrIndex));
    }
    return errors.map((e) => e.message);
}
exports.validateReadEventArgument = validateReadEventArgument;
//# sourceMappingURL=validateReadEventArgument.js.map