"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildModule = void 0;
const errors_1 = require("./errors");
const errors_list_1 = require("./internal/errors-list");
const module_builder_1 = require("./internal/module-builder");
const identifier_validators_1 = require("./internal/utils/identifier-validators");
/**
 * Construct a module definition that can be deployed through Ignition.
 *
 * @param moduleId - the id of the module
 * @param moduleDefintionFunction - a function accepting the
 * IgnitionModuleBuilder to configure the deployment
 * @returns a module definition
 *
 * @beta
 */
function buildModule(moduleId, moduleDefintionFunction) {
    if (typeof moduleId !== "string") {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.MODULE.INVALID_MODULE_ID);
    }
    if (!(0, identifier_validators_1.isValidIgnitionIdentifier)(moduleId)) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.MODULE.INVALID_MODULE_ID_CHARACTERS, {
            moduleId,
        });
    }
    if (typeof moduleDefintionFunction !== "function") {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.MODULE.INVALID_MODULE_DEFINITION_FUNCTION);
    }
    const constructor = new module_builder_1.ModuleConstructor();
    const ignitionModule = constructor.construct({
        id: moduleId,
        moduleDefintionFunction,
    });
    _checkForDuplicateModuleIds(ignitionModule);
    return ignitionModule;
}
exports.buildModule = buildModule;
/**
 * Check to ensure that there are no duplicate module ids among the root
 * module and its submodules.
 */
function _checkForDuplicateModuleIds(ignitionModule) {
    const duplicateModuleIds = [
        ignitionModule.id,
        ...Array.from(ignitionModule.submodules).map((submodule) => submodule.id),
    ].filter((id, index, array) => array.indexOf(id) !== index);
    if (duplicateModuleIds.length === 0) {
        return;
    }
    throw new errors_1.IgnitionError(errors_list_1.ERRORS.MODULE.DUPLICATE_MODULE_ID, {
        duplicateModuleIds: duplicateModuleIds.join(", "),
    });
}
//# sourceMappingURL=build-module.js.map