import { VarsManager } from "./vars-manager";
/**
 * This class is ONLY used when collecting the required and optional vars that have to be filled by the user
 */
export declare class VarsManagerSetup extends VarsManager {
    private readonly _getVarsAlreadySet;
    private readonly _hasVarsAlreadySet;
    private readonly _getVarsWithDefaultValueAlreadySet;
    private readonly _getVarsToSet;
    private readonly _hasVarsToSet;
    private readonly _getVarsWithDefaultValueToSet;
    constructor(varsFilePath: string);
    has(key: string): boolean;
    get(key: string, defaultValue?: string): string;
    getRequiredVarsAlreadySet(): string[];
    getOptionalVarsAlreadySet(): string[];
    getRequiredVarsToSet(): string[];
    getOptionalVarsToSet(): string[];
    private _getRequired;
    private _getOptionals;
}
//# sourceMappingURL=vars-manager-setup.d.ts.map