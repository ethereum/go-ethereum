interface Suggestion {
    name: string;
    description: string;
}
interface CompletionEnv {
    line: string;
    point: number;
}
export declare const HARDHAT_COMPLETE_FILES = "__hardhat_complete_files__";
export declare const REQUIRED_HH_VERSION_RANGE = "^1.0.0";
export declare function complete({ line, point, }: CompletionEnv): Promise<Suggestion[] | typeof HARDHAT_COMPLETE_FILES>;
export {};
//# sourceMappingURL=autocomplete.d.ts.map