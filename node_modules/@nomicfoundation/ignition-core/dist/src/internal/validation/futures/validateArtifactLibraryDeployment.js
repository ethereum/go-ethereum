"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateArtifactLibraryDeployment = void 0;
const type_guards_1 = require("../../../type-guards");
const libraries_1 = require("../../execution/libraries");
const utils_1 = require("../utils");
async function validateArtifactLibraryDeployment(future, _artifactLoader, _deploymentParameters, accounts) {
    const errors = [];
    /* stage two */
    if ((0, type_guards_1.isAccountRuntimeValue)(future.from)) {
        errors.push(...(0, utils_1.validateAccountRuntimeValue)(future.from, accounts));
    }
    errors.push(...(0, libraries_1.validateLibraryNames)(future.artifact, Object.keys(future.libraries)));
    return errors.map((e) => e.message);
}
exports.validateArtifactLibraryDeployment = validateArtifactLibraryDeployment;
//# sourceMappingURL=validateArtifactLibraryDeployment.js.map