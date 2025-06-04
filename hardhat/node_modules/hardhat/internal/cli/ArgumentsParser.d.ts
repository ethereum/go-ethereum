import { HardhatArguments, HardhatParamDefinitions, ScopesMap, TaskArguments, TaskDefinition, TasksMap } from "../../types";
export declare class ArgumentsParser {
    static readonly PARAM_PREFIX = "--";
    static paramNameToCLA(paramName: string): string;
    static cLAToParamName(cLA: string): string;
    parseHardhatArguments(hardhatParamDefinitions: HardhatParamDefinitions, envVariableArguments: HardhatArguments, rawCLAs: string[]): {
        hardhatArguments: HardhatArguments;
        scopeOrTaskName: string | undefined;
        allUnparsedCLAs: string[];
    };
    parseScopeAndTaskNames(allUnparsedCLAs: string[], taskDefinitions: TasksMap, scopeDefinitions: ScopesMap): {
        scopeName?: string;
        taskName: string;
        unparsedCLAs: string[];
    };
    parseTaskArguments(taskDefinition: TaskDefinition, rawCLAs: string[]): TaskArguments;
    private _parseTaskParamArguments;
    private _addHardhatDefaultArguments;
    private _addTaskDefaultArguments;
    private _isCLAParamName;
    private _hasCLAParamNameFormat;
    private _parseArgumentAt;
    private _parsePositionalParamArgs;
}
//# sourceMappingURL=ArgumentsParser.d.ts.map