"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const debug_1 = __importDefault(require("debug"));
const context_1 = require("../context");
const config_loading_1 = require("../core/config/config-loading");
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
const env_variables_1 = require("../core/params/env-variables");
const hardhat_params_1 = require("../core/params/hardhat-params");
const runtime_environment_1 = require("../core/runtime-environment");
let ctx;
let env;
if (context_1.HardhatContext.isCreated()) {
    ctx = context_1.HardhatContext.getHardhatContext();
    // The most probable reason for this to happen is that this file was imported
    // from the config file
    if (ctx.environment === undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.LIB_IMPORTED_FROM_THE_CONFIG);
    }
    env = ctx.environment;
}
else {
    ctx = context_1.HardhatContext.createHardhatContext();
    const hardhatArguments = (0, env_variables_1.getEnvHardhatArguments)(hardhat_params_1.HARDHAT_PARAM_DEFINITIONS, process.env);
    if (hardhatArguments.verbose) {
        debug_1.default.enable("hardhat*");
    }
    const { resolvedConfig, userConfig } = (0, config_loading_1.loadConfigAndTasks)(hardhatArguments);
    env = new runtime_environment_1.Environment(resolvedConfig, hardhatArguments, ctx.tasksDSL.getTaskDefinitions(), ctx.tasksDSL.getScopesDefinitions(), ctx.environmentExtenders, userConfig, ctx.providerExtenders);
    ctx.setHardhatRuntimeEnvironment(env);
}
module.exports = env;
//# sourceMappingURL=hardhat-lib.js.map