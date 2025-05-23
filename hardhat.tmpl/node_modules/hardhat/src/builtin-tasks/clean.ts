import fsExtra from "fs-extra";

import { subtask, task } from "../internal/core/config/config-env";
import { getCacheDir } from "../internal/util/global-dir";

import { TASK_CLEAN, TASK_CLEAN_GLOBAL } from "./task-names";

subtask(TASK_CLEAN_GLOBAL, async () => {
  const globalCacheDir = await getCacheDir();
  await fsExtra.emptyDir(globalCacheDir);
});

task(TASK_CLEAN, "Clears the cache and deletes all artifacts")
  .addFlag("global", "Clear the global cache")
  .setAction(
    async ({ global }: { global: boolean }, { config, run, artifacts }) => {
      if (global) {
        return run(TASK_CLEAN_GLOBAL);
      }
      await fsExtra.emptyDir(config.paths.cache);
      await fsExtra.remove(config.paths.artifacts);
      artifacts.clearCache?.();
    }
  );
