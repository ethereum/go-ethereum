"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.VarsManager = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const debug_1 = __importDefault(require("debug"));
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const log = (0, debug_1.default)("hardhat:core:vars:varsManager");
class VarsManager {
    constructor(_varsFilePath) {
        this._varsFilePath = _varsFilePath;
        this._VERSION = "hh-vars-1";
        this._ENV_VAR_PREFIX = "HARDHAT_VAR_";
        log("Creating a new instance of VarsManager");
        this._initializeVarsFile();
        this._storageCache = fs_extra_1.default.readJSONSync(this._varsFilePath);
        this._envCache = {};
        this._loadVarsFromEnv();
    }
    getStoragePath() {
        return this._varsFilePath;
    }
    set(key, value) {
        this.validateKey(key);
        if (value === "") {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.INVALID_EMPTY_VALUE);
        }
        const vars = this._storageCache.vars;
        vars[key] = { value };
        this._writeStoredVars(vars);
    }
    has(key, includeEnvs = false) {
        if (includeEnvs && key in this._envCache) {
            return true;
        }
        return key in this._storageCache.vars;
    }
    get(key, defaultValue, includeEnvs = false) {
        if (includeEnvs && key in this._envCache) {
            return this._envCache[key];
        }
        return this._storageCache.vars[key]?.value ?? defaultValue;
    }
    getEnvVars() {
        return Object.keys(this._envCache).map((k) => `${this._ENV_VAR_PREFIX}${k}`);
    }
    list() {
        return Object.keys(this._storageCache.vars);
    }
    delete(key) {
        const vars = this._storageCache.vars;
        if (vars[key] === undefined)
            return false;
        delete vars[key];
        this._writeStoredVars(vars);
        return true;
    }
    validateKey(key) {
        const KEY_REGEX = /^[a-zA-Z_]+[a-zA-Z0-9_]*$/;
        if (!KEY_REGEX.test(key)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.INVALID_CONFIG_VAR_NAME, {
                value: key,
            });
        }
    }
    _initializeVarsFile() {
        if (!fs_extra_1.default.pathExistsSync(this._varsFilePath)) {
            // Initialize the vars file if it does not exist
            log(`Vars file do not exist. Creating a new one at '${this._varsFilePath}' with version '${this._VERSION}'`);
            fs_extra_1.default.writeJSONSync(this._varsFilePath, this._getVarsFileStructure(), {
                spaces: 2,
            });
        }
    }
    _getVarsFileStructure() {
        return {
            _format: this._VERSION,
            vars: {},
        };
    }
    _loadVarsFromEnv() {
        log("Loading ENV variables if any");
        for (const key in process.env) {
            if (key.startsWith(this._ENV_VAR_PREFIX)) {
                const envVar = process.env[key];
                if (envVar === undefined ||
                    envVar.replace(/[\s\t]/g, "").length === 0) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_ENV_VAR_VALUE, {
                        varName: key,
                        value: envVar,
                    });
                }
                const envKey = key.replace(this._ENV_VAR_PREFIX, "");
                this.validateKey(envKey);
                // Store only in cache, not in a file, as the vars are sourced from environment variables
                this._envCache[envKey] = envVar;
            }
        }
    }
    _writeStoredVars(vars) {
        // ENV variables are not stored in the file
        this._storageCache.vars = vars;
        fs_extra_1.default.writeJSONSync(this._varsFilePath, this._storageCache, { spaces: 2 });
    }
}
exports.VarsManager = VarsManager;
//# sourceMappingURL=vars-manager.js.map