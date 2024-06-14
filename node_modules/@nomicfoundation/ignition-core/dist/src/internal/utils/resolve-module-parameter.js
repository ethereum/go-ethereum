"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveModuleParameter = void 0;
const type_guards_1 = require("../../type-guards");
const future_resolvers_1 = require("../execution/future-processor/helpers/future-resolvers");
const assertions_1 = require("./assertions");
function resolveModuleParameter(moduleParamRuntimeValue, context) {
    if (context.deploymentParameters === undefined) {
        (0, assertions_1.assertIgnitionInvariant)(moduleParamRuntimeValue.defaultValue !== undefined, `No default value provided for module parameter ${moduleParamRuntimeValue.moduleId}/${moduleParamRuntimeValue.name}`);
        return _resolveDefaultValue(moduleParamRuntimeValue, context.accounts);
    }
    const moduleParameters = context.deploymentParameters[moduleParamRuntimeValue.moduleId];
    if (moduleParameters === undefined) {
        (0, assertions_1.assertIgnitionInvariant)(moduleParamRuntimeValue.defaultValue !== undefined, `No default value provided for module parameter ${moduleParamRuntimeValue.moduleId}/${moduleParamRuntimeValue.name}`);
        return _resolveDefaultValue(moduleParamRuntimeValue, context.accounts);
    }
    const moduleParamValue = moduleParameters[moduleParamRuntimeValue.name];
    if (moduleParamValue === undefined) {
        return _resolveDefaultValue(moduleParamRuntimeValue, context.accounts);
    }
    return moduleParamValue;
}
exports.resolveModuleParameter = resolveModuleParameter;
function _resolveDefaultValue(moduleParamRuntimeValue, accounts) {
    (0, assertions_1.assertIgnitionInvariant)(moduleParamRuntimeValue.defaultValue !== undefined, `No default value provided for module parameter ${moduleParamRuntimeValue.moduleId}/${moduleParamRuntimeValue.name}`);
    if ((0, type_guards_1.isAccountRuntimeValue)(moduleParamRuntimeValue.defaultValue)) {
        return (0, future_resolvers_1.resolveAccountRuntimeValue)(moduleParamRuntimeValue.defaultValue, accounts);
    }
    return moduleParamRuntimeValue.defaultValue;
}
//# sourceMappingURL=resolve-module-parameter.js.map