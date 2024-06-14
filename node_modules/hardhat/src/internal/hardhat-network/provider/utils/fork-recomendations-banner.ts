import chalk from "chalk";
import fsExtra from "fs-extra";
import path from "path";

import { NetworkConfig } from "../../../../types";

function getAlreadyShownFilePath(forkCachePath: string) {
  return path.join(forkCachePath, "recommendations-already-shown.json");
}

function displayBanner() {
  console.warn(
    chalk.yellow(
      `You're running a network fork starting from the latest block.
Performance may degrade due to fetching data from the network with each run.
If connecting to an archival node (e.g. Alchemy), we strongly recommend setting
blockNumber to a fixed value to increase performance with a local cache.`
    )
  );
}

export async function showForkRecommendationsBannerIfNecessary(
  currentNetworkConfig: NetworkConfig,
  forkCachePath: string
) {
  if (!("forking" in currentNetworkConfig)) {
    return;
  }

  if (currentNetworkConfig.forking?.enabled !== true) {
    return;
  }

  if (currentNetworkConfig.forking?.blockNumber !== undefined) {
    return;
  }

  const shownPath = getAlreadyShownFilePath(forkCachePath);

  if (await fsExtra.pathExists(shownPath)) {
    return;
  }

  displayBanner();

  await fsExtra.ensureDir(path.dirname(shownPath));
  await fsExtra.writeJSON(shownPath, true);
}
