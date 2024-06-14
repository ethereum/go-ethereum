import { HardhatArguments, HardhatConfig, HardhatUserConfig, SolcConfig } from "../../../types";
export declare function importCsjOrEsModule(filePath: string): any;
export declare function resolveConfigPath(configPath: string | undefined): string;
export declare function loadConfigAndTasks(hardhatArguments?: Partial<HardhatArguments>, { showEmptyConfigWarning, showSolidityConfigWarnings, }?: {
    showEmptyConfigWarning?: boolean;
    showSolidityConfigWarnings?: boolean;
}): {
    resolvedConfig: HardhatConfig;
    userConfig: HardhatUserConfig;
};
/**
 * Receives an Error and checks if it's a MODULE_NOT_FOUND and the reason that
 * caused it.
 *
 * If it can infer the reason, it throws an appropiate error. Otherwise it does
 * nothing.
 */
export declare function analyzeModuleNotFoundError(error: any, configPath: string): void;
export declare function getConfiguredCompilers(solidityConfig: HardhatConfig["solidity"]): SolcConfig[];
//# sourceMappingURL=config-loading.d.ts.map