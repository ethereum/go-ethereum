import findUp from "find-up";
import fsExtra from "fs-extra";
import path from "path";

import { getPackageRoot } from "../util/packageInfo";

import { HardhatError } from "./errors";
import { ERRORS } from "./errors-list";

const JS_CONFIG_FILENAME = "hardhat.config.js";
const CJS_CONFIG_FILENAME = "hardhat.config.cjs";
const TS_CONFIG_FILENAME = "hardhat.config.ts";
const CTS_CONFIG_FILENAME = "hardhat.config.cts";

export function isCwdInsideProject() {
  return (
    findUp.sync(TS_CONFIG_FILENAME) !== undefined ||
    findUp.sync(CTS_CONFIG_FILENAME) !== undefined ||
    findUp.sync(CJS_CONFIG_FILENAME) !== undefined ||
    findUp.sync(JS_CONFIG_FILENAME) !== undefined
  );
}

export function getUserConfigPath() {
  const tsConfigPath = findUp.sync(TS_CONFIG_FILENAME);
  if (tsConfigPath !== undefined) {
    return tsConfigPath;
  }

  const ctsConfigPath = findUp.sync(CTS_CONFIG_FILENAME);
  if (ctsConfigPath !== undefined) {
    return ctsConfigPath;
  }

  const cjsConfigPath = findUp.sync(CJS_CONFIG_FILENAME);
  if (cjsConfigPath !== undefined) {
    return cjsConfigPath;
  }

  const pathToConfigFile = findUp.sync(JS_CONFIG_FILENAME);
  if (pathToConfigFile === undefined) {
    throw new HardhatError(ERRORS.GENERAL.NOT_INSIDE_PROJECT);
  }

  return pathToConfigFile;
}

export async function getRecommendedGitIgnore() {
  const packageRoot = getPackageRoot();
  const gitIgnorePath = path.join(packageRoot, "recommended-gitignore.txt");

  return fsExtra.readFile(gitIgnorePath, "utf-8");
}
