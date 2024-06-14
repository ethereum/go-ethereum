import { HardhatArguments } from "../../types";
export declare function runScript(scriptPath: string, scriptArgs?: string[], extraNodeArgs?: string[], extraEnvVars?: {
    [name: string]: string;
}): Promise<number>;
export declare function runScriptWithHardhat(hardhatArguments: HardhatArguments, scriptPath: string, scriptArgs?: string[], extraNodeArgs?: string[], extraEnvVars?: {
    [name: string]: string;
}): Promise<number>;
//# sourceMappingURL=scripts-runner.d.ts.map