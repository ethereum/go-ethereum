"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.loadModule = void 0;
const ignition_core_1 = require("@nomicfoundation/ignition-core");
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = require("fs-extra");
const plugins_1 = require("hardhat/plugins");
const path_1 = __importDefault(require("path"));
const shouldBeHardhatPluginError_1 = require("./shouldBeHardhatPluginError");
const debug = (0, debug_1.default)("hardhat-ignition:modules");
const MODULES_FOLDER = "modules";
function loadModule(ignitionDirectory, modulePath) {
    const fullModulesDirectoryName = path_1.default.resolve(ignitionDirectory, MODULES_FOLDER);
    const shortModulesDirectoryName = path_1.default.join(ignitionDirectory, MODULES_FOLDER);
    debug(`Loading user modules from '${fullModulesDirectoryName}'`);
    const fullpathToModule = path_1.default.resolve(modulePath);
    if (!(0, fs_extra_1.pathExistsSync)(fullpathToModule)) {
        throw new plugins_1.HardhatPluginError("hardhat-ignition", `Could not find a module file at the path: ${modulePath}`);
    }
    if (!isInModuleDirectory(fullModulesDirectoryName, fullpathToModule)) {
        throw new plugins_1.HardhatPluginError("hardhat-ignition", `The referenced module file ${modulePath} is outside the module directory ${shortModulesDirectoryName}`);
    }
    debug(`Loading module file '${fullpathToModule}'`);
    let module;
    try {
        module = require(fullpathToModule);
    }
    catch (e) {
        if (e instanceof ignition_core_1.IgnitionError) {
            /**
             * Errors thrown from within ModuleBuilder use this errorNumber.
             *
             * They have a stack trace that's useful to the user, so we display it here, instead of
             * wrapping the error in a NomicLabsHardhatPluginError.
             */
            if (e.errorNumber === 702) {
                console.error(e);
                throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", "Module validation failed. Check the stack trace above to identify the issue and its source code location.");
            }
            if ((0, shouldBeHardhatPluginError_1.shouldBeHardhatPluginError)(e)) {
                throw new plugins_1.NomicLabsHardhatPluginError("hardhat-ignition", e.message, e);
            }
        }
        throw e;
    }
    return module.default ?? module;
}
exports.loadModule = loadModule;
function isInModuleDirectory(modulesDirectory, modulePath) {
    const resolvedModulesDirectory = path_1.default.resolve(modulesDirectory);
    const moduleRelativeToModuleDir = path_1.default.relative(resolvedModulesDirectory, modulePath);
    return (!moduleRelativeToModuleDir.startsWith("..") &&
        !path_1.default.isAbsolute(moduleRelativeToModuleDir));
}
//# sourceMappingURL=load-module.js.map