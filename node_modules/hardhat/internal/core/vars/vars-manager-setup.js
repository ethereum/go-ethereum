"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.VarsManagerSetup = void 0;
const debug_1 = __importDefault(require("debug"));
const vars_manager_1 = require("./vars-manager");
const log = (0, debug_1.default)("hardhat:core:vars:varsManagerSetup");
/**
 * This class is ONLY used when collecting the required and optional vars that have to be filled by the user
 */
class VarsManagerSetup extends vars_manager_1.VarsManager {
    constructor(varsFilePath) {
        log("Creating a new instance of VarsManagerSetup");
        super(varsFilePath);
        this._getVarsAlreadySet = new Set();
        this._hasVarsAlreadySet = new Set();
        this._getVarsWithDefaultValueAlreadySet = new Set();
        this._getVarsToSet = new Set();
        this._hasVarsToSet = new Set();
        this._getVarsWithDefaultValueToSet = new Set();
    }
    // Checks if the key exists, and updates sets accordingly.
    // Ignore the parameter 'includeEnvs' defined in the parent class because during setup env vars are ignored.
    has(key) {
        log(`function 'has' called with key '${key}'`);
        const hasKey = super.has(key);
        if (hasKey) {
            this._hasVarsAlreadySet.add(key);
        }
        else {
            this._hasVarsToSet.add(key);
        }
        return hasKey;
    }
    // Gets the value for the provided key, and updates sets accordingly.
    // Ignore the parameter 'includeEnvs' defined in the parent class because during setup env vars are ignored.
    get(key, defaultValue) {
        log(`function 'get' called with key '${key}'`);
        const varAlreadySet = super.has(key);
        if (varAlreadySet) {
            if (defaultValue !== undefined) {
                this._getVarsWithDefaultValueAlreadySet.add(key);
            }
            else {
                this._getVarsAlreadySet.add(key);
            }
        }
        else {
            if (defaultValue !== undefined) {
                this._getVarsWithDefaultValueToSet.add(key);
            }
            else {
                this._getVarsToSet.add(key);
            }
        }
        // Do not return undefined to avoid throwing an error
        return super.get(key, defaultValue) ?? "";
    }
    getRequiredVarsAlreadySet() {
        return this._getRequired(this._getVarsAlreadySet, this._hasVarsAlreadySet);
    }
    getOptionalVarsAlreadySet() {
        return this._getOptionals(this._getVarsAlreadySet, this._hasVarsAlreadySet, this._getVarsWithDefaultValueAlreadySet);
    }
    getRequiredVarsToSet() {
        return this._getRequired(this._getVarsToSet, this._hasVarsToSet);
    }
    getOptionalVarsToSet() {
        return this._getOptionals(this._getVarsToSet, this._hasVarsToSet, this._getVarsWithDefaultValueToSet);
    }
    // How to calculate required and optional variables:
    //
    // G = get function
    // H = has function
    // GD = get function with default value
    //
    // optional variables = H + (GD - G)
    // required variables = G - H
    _getRequired(getVars, hasVars) {
        return Array.from(getVars).filter((k) => !hasVars.has(k));
    }
    _getOptionals(getVars, hasVars, getVarsWithDefault) {
        const result = new Set(hasVars);
        for (const k of getVarsWithDefault) {
            if (!getVars.has(k)) {
                result.add(k);
            }
        }
        return Array.from(result);
    }
}
exports.VarsManagerSetup = VarsManagerSetup;
//# sourceMappingURL=vars-manager-setup.js.map