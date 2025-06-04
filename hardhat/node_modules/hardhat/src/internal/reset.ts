import { HardhatContext } from "./context";

// This function isn't meant to be used during the Hardhat execution,
// but rather to reset Hardhat in between tests.
export function resetHardhatContext() {
  if (HardhatContext.isCreated()) {
    const ctx = HardhatContext.getHardhatContext();

    if (ctx.environment !== undefined) {
      const globalAsAny = global as any;
      for (const key of Object.keys(ctx.environment)) {
        globalAsAny.hre = undefined;
        globalAsAny[key] = undefined;
      }
    }

    const filesLoadedDuringConfig = ctx.getFilesLoadedDuringConfig();
    filesLoadedDuringConfig.forEach(unloadModule);

    HardhatContext.deleteHardhatContext();
  }

  // Unload all the hardhat's entry-points.
  unloadModule("../register");
  unloadModule("./cli/cli");
  unloadModule("./lib/hardhat-lib");
}

function unloadModule(path: string) {
  try {
    delete require.cache[require.resolve(path)];
  } catch {
    // module wasn't loaded
  }
}
