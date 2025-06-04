"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const HelpPrinter_1 = require("../internal/cli/HelpPrinter");
const constants_1 = require("../internal/constants");
const config_env_1 = require("../internal/core/config/config-env");
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
const hardhat_params_1 = require("../internal/core/params/hardhat-params");
const task_names_1 = require("./task-names");
(0, config_env_1.task)(task_names_1.TASK_HELP, "Prints this message")
    .addOptionalPositionalParam("scopeOrTask", "An optional scope or task to print more info about")
    .addOptionalPositionalParam("task", "An optional task to print more info about")
    .setAction(async ({ scopeOrTask, task: taskName }, { tasks, scopes, version }) => {
    const helpPrinter = new HelpPrinter_1.HelpPrinter(constants_1.HARDHAT_NAME, constants_1.HARDHAT_EXECUTABLE_NAME, version, hardhat_params_1.HARDHAT_PARAM_DEFINITIONS, tasks, scopes);
    if (scopeOrTask === undefined) {
        // no params, print global help
        helpPrinter.printGlobalHelp();
        return;
    }
    const taskDefinition = tasks[scopeOrTask];
    if (taskDefinition !== undefined) {
        // the first param is a valid task
        helpPrinter.printTaskHelp(tasks[scopeOrTask]);
        return;
    }
    const scopeDefinition = scopes[scopeOrTask];
    if (scopeDefinition === undefined) {
        // if the first parameter is neither a task nor a scope,
        // we don't know what the user was trying to print,
        // so we assume that it's an unrecognized task
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
            task: scopeOrTask,
        });
    }
    if (taskName === undefined) {
        // if the second parameter is not present, print scope help
        helpPrinter.printScopeHelp(scopeDefinition);
        return;
    }
    const scopedTaskDefinition = scopeDefinition.tasks[taskName];
    if (scopedTaskDefinition === undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
            scope: scopeOrTask,
            task: taskName,
        });
    }
    helpPrinter.printTaskHelp(scopedTaskDefinition);
});
//# sourceMappingURL=help.js.map