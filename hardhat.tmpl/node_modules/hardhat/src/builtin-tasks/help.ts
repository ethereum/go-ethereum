import { HelpPrinter } from "../internal/cli/HelpPrinter";
import { HARDHAT_EXECUTABLE_NAME, HARDHAT_NAME } from "../internal/constants";
import { task } from "../internal/core/config/config-env";
import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";
import { HARDHAT_PARAM_DEFINITIONS } from "../internal/core/params/hardhat-params";

import { TASK_HELP } from "./task-names";

task(TASK_HELP, "Prints this message")
  .addOptionalPositionalParam(
    "scopeOrTask",
    "An optional scope or task to print more info about"
  )
  .addOptionalPositionalParam(
    "task",
    "An optional task to print more info about"
  )
  .setAction(
    async (
      { scopeOrTask, task: taskName }: { scopeOrTask?: string; task?: string },
      { tasks, scopes, version }
    ) => {
      const helpPrinter = new HelpPrinter(
        HARDHAT_NAME,
        HARDHAT_EXECUTABLE_NAME,
        version,
        HARDHAT_PARAM_DEFINITIONS,
        tasks,
        scopes
      );

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
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
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
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
          scope: scopeOrTask,
          task: taskName,
        });
      }

      helpPrinter.printTaskHelp(scopedTaskDefinition);
    }
  );
