import debug from "debug";
import fsExtra from "fs-extra";

import { task } from "../internal/core/config/config-env";
import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";
import { runScriptWithHardhat } from "../internal/util/scripts-runner";

import { TASK_COMPILE, TASK_RUN } from "./task-names";

const log = debug("hardhat:core:tasks:run");

task(TASK_RUN, "Runs a user-defined script after compiling the project")
  .addPositionalParam(
    "script",
    "A js file to be run within hardhat's environment"
  )
  .addFlag("noCompile", "Don't compile before running this task")
  .setAction(
    async (
      { script, noCompile }: { script: string; noCompile: boolean },
      { run, hardhatArguments }
    ) => {
      if (!(await fsExtra.pathExists(script))) {
        throw new HardhatError(ERRORS.BUILTIN_TASKS.RUN_FILE_NOT_FOUND, {
          script,
        });
      }

      if (!noCompile) {
        await run(TASK_COMPILE, { quiet: true });
      }

      log(
        `Running script ${script} in a subprocess so we can wait for it to complete`
      );

      try {
        process.exitCode = await runScriptWithHardhat(hardhatArguments, script);
      } catch (error) {
        if (error instanceof Error) {
          throw new HardhatError(
            ERRORS.BUILTIN_TASKS.RUN_SCRIPT_ERROR,
            {
              script,
              error: error.message,
            },
            error
          );
        }

        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
      }
    }
  );
