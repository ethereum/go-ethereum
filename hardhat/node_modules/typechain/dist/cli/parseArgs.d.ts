export interface ParsedArgs {
    files: string[];
    target: string;
    outDir?: string | undefined;
    inputDir?: string | undefined;
    flags: {
        discriminateTypes: boolean;
        alwaysGenerateOverloads: boolean;
        tsNocheck: boolean;
        node16Modules: boolean;
    };
}
export declare function parseArgs(): ParsedArgs;
