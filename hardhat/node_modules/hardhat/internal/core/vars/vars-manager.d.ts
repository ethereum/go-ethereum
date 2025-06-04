export declare class VarsManager {
    private readonly _varsFilePath;
    private readonly _VERSION;
    private readonly _ENV_VAR_PREFIX;
    private readonly _storageCache;
    private readonly _envCache;
    constructor(_varsFilePath: string);
    getStoragePath(): string;
    set(key: string, value: string): void;
    has(key: string, includeEnvs?: boolean): boolean;
    get(key: string, defaultValue?: string, includeEnvs?: boolean): string | undefined;
    getEnvVars(): string[];
    list(): string[];
    delete(key: string): boolean;
    validateKey(key: string): void;
    private _initializeVarsFile;
    private _getVarsFileStructure;
    private _loadVarsFromEnv;
    private _writeStoredVars;
}
//# sourceMappingURL=vars-manager.d.ts.map