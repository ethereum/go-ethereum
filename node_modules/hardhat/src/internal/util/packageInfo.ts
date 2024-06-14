import findup from "find-up";
import fsExtra from "fs-extra";
import path from "path";

import { assertHardhatInvariant } from "../core/errors";

export function getPackageJsonPath(): string {
  return findClosestPackageJson(__filename)!;
}

export function getPackageRoot(): string {
  const packageJsonPath = getPackageJsonPath();

  return path.dirname(packageJsonPath);
}

export interface PackageJson {
  name: string;
  version: string;
  type?: "commonjs" | "module";
  engines: {
    node: string;
  };
}

export function findClosestPackageJson(file: string): string | null {
  return findup.sync("package.json", { cwd: path.dirname(file) });
}

export async function getPackageName(file: string): Promise<string> {
  const packageJsonPath = findClosestPackageJson(file);
  if (packageJsonPath !== null && packageJsonPath !== "") {
    const packageJson: PackageJson = await fsExtra.readJSON(packageJsonPath);
    return packageJson.name;
  }
  return "";
}

export async function getPackageJson(): Promise<PackageJson> {
  const root = getPackageRoot();
  return fsExtra.readJSON(path.join(root, "package.json"));
}

export function getHardhatVersion(): string {
  const packageJsonPath = findClosestPackageJson(__filename);

  assertHardhatInvariant(
    packageJsonPath !== null,
    "There should be a package.json in hardhat-core's root directory"
  );

  const packageJson = fsExtra.readJsonSync(packageJsonPath);
  return packageJson.version;
}

/**
 * Return the contents of the package.json in the user's project
 */
export function getProjectPackageJson(): Promise<any> {
  const packageJsonPath = findup.sync("package.json");

  assertHardhatInvariant(
    packageJsonPath !== null,
    "Expected a package.json file in the current directory or in an ancestor directory"
  );

  return fsExtra.readJson(packageJsonPath);
}
