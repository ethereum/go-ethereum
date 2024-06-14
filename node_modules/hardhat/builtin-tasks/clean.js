"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const fs_extra_1 = __importDefault(require("fs-extra"));
const config_env_1 = require("../internal/core/config/config-env");
const global_dir_1 = require("../internal/util/global-dir");
const task_names_1 = require("./task-names");
(0, config_env_1.subtask)(task_names_1.TASK_CLEAN_GLOBAL, async () => {
    const globalCacheDir = await (0, global_dir_1.getCacheDir)();
    await fs_extra_1.default.emptyDir(globalCacheDir);
});
(0, config_env_1.task)(task_names_1.TASK_CLEAN, "Clears the cache and deletes all artifacts")
    .addFlag("global", "Clear the global cache")
    .setAction(async ({ global }, { config, run, artifacts }) => {
    if (global) {
        return run(task_names_1.TASK_CLEAN_GLOBAL);
    }
    await fs_extra_1.default.emptyDir(config.paths.cache);
    await fs_extra_1.default.remove(config.paths.artifacts);
    artifacts.clearCache?.();
});
//# sourceMappingURL=clean.js.map