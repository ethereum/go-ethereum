"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HardhatContext = void 0;
const errors_1 = require("./core/errors");
const errors_list_1 = require("./core/errors-list");
const vars_manager_1 = require("./core/vars/vars-manager");
const dsl_1 = require("./core/tasks/dsl");
const global_dir_1 = require("./util/global-dir");
const platform_1 = require("./util/platform");
class HardhatContext {
    constructor() {
        this.tasksDSL = new dsl_1.TasksDSL();
        this.environmentExtenders = [];
        this.providerExtenders = [];
        this.configExtenders = [];
        this.varsManager = new vars_manager_1.VarsManager((0, global_dir_1.getVarsFilePath)());
    }
    static isCreated() {
        const globalWithHardhatContext = global;
        return globalWithHardhatContext.__hardhatContext !== undefined;
    }
    static createHardhatContext() {
        if (this.isCreated()) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.CONTEXT_ALREADY_CREATED);
        }
        const globalWithHardhatContext = global;
        const ctx = new HardhatContext();
        globalWithHardhatContext.__hardhatContext = ctx;
        return ctx;
    }
    static getHardhatContext() {
        const globalWithHardhatContext = global;
        const ctx = globalWithHardhatContext.__hardhatContext;
        if (ctx === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.CONTEXT_NOT_CREATED);
        }
        return ctx;
    }
    static deleteHardhatContext() {
        const globalAsAny = global;
        globalAsAny.__hardhatContext = undefined;
    }
    setHardhatRuntimeEnvironment(env) {
        if (this.environment !== undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.CONTEXT_HRE_ALREADY_DEFINED);
        }
        this.environment = env;
    }
    getHardhatRuntimeEnvironment() {
        if (this.environment === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.CONTEXT_HRE_NOT_DEFINED);
        }
        return this.environment;
    }
    setConfigLoadingAsStarted() {
        this._filesLoadedBeforeConfig = (0, platform_1.getRequireCachedFiles)();
    }
    setConfigLoadingAsFinished() {
        this._filesLoadedAfterConfig = (0, platform_1.getRequireCachedFiles)();
    }
    getFilesLoadedDuringConfig() {
        // No config was loaded
        if (this._filesLoadedBeforeConfig === undefined) {
            return [];
        }
        (0, errors_1.assertHardhatInvariant)(this._filesLoadedAfterConfig !== undefined, "Config loading was set as started and not finished");
        return arraysDifference(this._filesLoadedAfterConfig, this._filesLoadedBeforeConfig);
    }
}
exports.HardhatContext = HardhatContext;
function arraysDifference(a, b) {
    return a.filter((e) => !b.includes(e));
}
//# sourceMappingURL=context.js.map