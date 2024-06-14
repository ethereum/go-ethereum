import { HardhatConfig, HardhatUserConfig, ProjectPathsConfig, ProjectPathsUserConfig } from "../../../types";
/**
 * This functions resolves the hardhat config, setting its defaults and
 * normalizing its types if necessary.
 *
 * @param userConfigPath the user config filepath
 * @param userConfig     the user config object
 *
 * @returns the resolved config
 */
export declare function resolveConfig(userConfigPath: string, userConfig: HardhatUserConfig): HardhatConfig;
/**
 * This function resolves the ProjectPathsConfig object from the user-provided config
 * and its path. The logic of this is not obvious and should well be document.
 * The good thing is that most users will never use this.
 *
 * Explanation:
 *    - paths.configFile is not overridable
 *    - If a path is absolute it is used "as is".
 *    - If the root path is relative, it's resolved from paths.configFile's dir.
 *    - If any other path is relative, it's resolved from paths.root.
 *    - Plugin-defined paths are not resolved, but encouraged to follow the same pattern.
 */
export declare function resolveProjectPaths(userConfigPath: string, userPaths?: ProjectPathsUserConfig): ProjectPathsConfig;
//# sourceMappingURL=config-resolution.d.ts.map