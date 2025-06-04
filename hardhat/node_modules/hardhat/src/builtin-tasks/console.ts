import debug from "debug";
import fsExtra from "fs-extra";
import * as path from "path";
import * as semver from "semver";

import { task } from "../internal/core/config/config-env";
import { runScriptWithHardhat } from "../internal/util/scripts-runner";

import { TASK_COMPILE, TASK_CONSOLE } from "./task-names";

const log = debug("hardhat:core:tasks:console");

task(TASK_CONSOLE, "Opens a hardhat console")
  .addFlag("noCompile", "Don't compile before running this task")
  .setAction(
    async (
      { noCompile }: { noCompile: boolean },
      { config, run, hardhatArguments }
    ) => {
      if (!noCompile) {
        await run(TASK_COMPILE, { quiet: true });
      }

      await fsExtra.ensureDir(config.paths.cache);
      const historyFile = path.join(config.paths.cache, "console-history.txt");

      const nodeArgs = [];
      if (semver.gte(process.version, "10.0.0")) {
        nodeArgs.push("--experimental-repl-await");
      }

      log(
        `Creating a Node REPL subprocess with Hardhat's register so we can set some Node's flags`
      );

      // Running the script "" is like running `node`, so this starts the repl
      await runScriptWithHardhat(hardhatArguments, "", [], nodeArgs, {
        NODE_REPL_HISTORY: historyFile,
      });
    }
  );
