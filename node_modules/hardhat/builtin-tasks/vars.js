"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const errors_1 = require("../internal/core/errors");
const config_env_1 = require("../internal/core/config/config-env");
const errors_list_1 = require("../internal/core/errors-list");
const varsScope = (0, config_env_1.scope)("vars", "Manage your configuration variables");
varsScope
    .task("set", "Set the value of a configuration variable")
    .addPositionalParam("var", "The name of the variable")
    .addOptionalPositionalParam("value", "The value to store. Omit to be prompted for it.")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
varsScope
    .task("get", "Get the value of a configuration variable")
    .addPositionalParam("var", "The name of the variable")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
varsScope
    .task("list", "List all the configuration variables")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
varsScope
    .task("delete", "Delete a configuration variable")
    .addPositionalParam("var", "The name of the variable")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
varsScope
    .task("path", "Show the path of the file where all the configuration variables are stored")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
varsScope
    .task("setup", "Show how to setup the configuration variables used by this project")
    .setAction(async () => {
    throw new errors_1.HardhatError(errors_list_1.ERRORS.VARS.ONLY_MANAGED_IN_CLI);
});
//# sourceMappingURL=vars.js.map