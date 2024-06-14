/**
 * Returns true if Hardhat is installed locally or linked from its repository,
 * by looking for it using the node module resolution logic.
 *
 * If a config file is provided, we start looking for it from it. Otherwise,
 * we use the current working directory.
 */
export declare function isHardhatInstalledLocallyOrLinked(configPath?: string): boolean;
/**
 * Checks whether we're using Hardhat in development mode (that is, we're working _on_ Hardhat).
 */
export declare function isLocalDev(): boolean;
export declare function isRunningHardhatCoreTests(): boolean;
//# sourceMappingURL=execution-mode.d.ts.map