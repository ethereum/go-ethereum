import path from "path";

import { ProjectPathsConfig } from "../../../../types";

export function getForkCacheDirPath(paths: ProjectPathsConfig): string {
  return path.join(paths.cache, "hardhat-network-fork");
}
