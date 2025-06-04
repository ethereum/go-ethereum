import { HardhatConfig } from "../../types";
/**
 * Returns true if Hardhat will run in using typescript mode.
 * @param configPath The config path if provider by the user.
 */
export declare function willRunWithTypescript(configPath?: string): boolean;
/**
 * Returns true if a Hardhat is already running with typescript.
 */
export declare function isRunningWithTypescript(config: HardhatConfig): boolean;
export declare function isTypescriptSupported(): boolean;
export declare function loadTsNode(tsConfigPath?: string, shouldTypecheck?: boolean): void;
export declare function isTypescriptFile(path: string): boolean;
export declare function isJavascriptFile(path: string): boolean;
//# sourceMappingURL=typescript-support.d.ts.map