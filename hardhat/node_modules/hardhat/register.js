"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const debug_1 = __importDefault(require("debug"));
const context_1 = require("./internal/context");
const config_loading_1 = require("./internal/core/config/config-loading");
const env_variables_1 = require("./internal/core/params/env-variables");
const hardhat_params_1 = require("./internal/core/params/hardhat-params");
const runtime_environment_1 = require("./internal/core/runtime-environment");
const typescript_support_1 = require("./internal/core/typescript-support");
const console_1 = require("./internal/util/console");
if (!context_1.HardhatContext.isCreated()) {
    require("source-map-support/register");
    const ctx = context_1.HardhatContext.createHardhatContext();
    if ((0, console_1.isNodeCalledWithoutAScript)()) {
        (0, console_1.disableReplWriterShowProxy)();
    }
    const hardhatArguments = (0, env_variables_1.getEnvHardhatArguments)(hardhat_params_1.HARDHAT_PARAM_DEFINITIONS, process.env);
    if (hardhatArguments.verbose) {
        debug_1.default.enable("hardhat*");
    }
    if ((0, typescript_support_1.willRunWithTypescript)(hardhatArguments.config)) {
        (0, typescript_support_1.loadTsNode)(hardhatArguments.tsconfig, hardhatArguments.typecheck);
    }
    const { resolvedConfig, userConfig } = (0, config_loading_1.loadConfigAndTasks)(hardhatArguments);
    const env = new runtime_environment_1.Environment(resolvedConfig, hardhatArguments, ctx.tasksDSL.getTaskDefinitions(), ctx.tasksDSL.getScopesDefinitions(), ctx.environmentExtenders, userConfig, ctx.providerExtenders);
    ctx.setHardhatRuntimeEnvironment(env);
    env.injectToGlobal();
}
//# sourceMappingURL=register.js.map