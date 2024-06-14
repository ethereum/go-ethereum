import { HardhatParamDefinitions, ScopeDefinition, ScopesMap, TaskDefinition, TasksMap } from "../../types";
export declare class HelpPrinter {
    private readonly _programName;
    private readonly _executableName;
    private readonly _version;
    private readonly _hardhatParamDefinitions;
    private readonly _tasks;
    private readonly _scopes;
    constructor(_programName: string, _executableName: string, _version: string, _hardhatParamDefinitions: HardhatParamDefinitions, _tasks: TasksMap, _scopes: ScopesMap);
    printGlobalHelp(includeSubtasks?: boolean): void;
    printScopeHelp(scopeDefinition: ScopeDefinition, includeSubtasks?: boolean): void;
    printTaskHelp(taskDefinition: TaskDefinition): void;
    private _printTasks;
    private _printScopes;
    private _getParamValueDescription;
    private _getParamsList;
    private _getPositionalParamsList;
    private _printParamDetails;
    private _printPositionalParamDetails;
}
//# sourceMappingURL=HelpPrinter.d.ts.map