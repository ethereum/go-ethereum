import {
  HardhatParamDefinitions,
  ParamDefinition,
  ParamDefinitionsMap,
  ScopeDefinition,
  ScopesMap,
  TaskDefinition,
  TasksMap,
} from "../../types";
import { HardhatError } from "../core/errors";
import { ERRORS } from "../core/errors-list";

import { ArgumentsParser } from "./ArgumentsParser";

export class HelpPrinter {
  constructor(
    private readonly _programName: string,
    private readonly _executableName: string,
    private readonly _version: string,
    private readonly _hardhatParamDefinitions: HardhatParamDefinitions,
    private readonly _tasks: TasksMap,
    private readonly _scopes: ScopesMap
  ) {}

  public printGlobalHelp(includeSubtasks = false) {
    console.log(`${this._programName} version ${this._version}\n`);

    console.log(
      `Usage: ${this._executableName} [GLOBAL OPTIONS] [SCOPE] <TASK> [TASK OPTIONS]\n`
    );

    console.log("GLOBAL OPTIONS:\n");

    let length = this._printParamDetails(this._hardhatParamDefinitions);

    console.log("\n\nAVAILABLE TASKS:\n");

    length = this._printTasks(this._tasks, includeSubtasks, length);

    if (Object.keys(this._scopes).length > 0) {
      console.log("\n\nAVAILABLE TASK SCOPES:\n");

      this._printScopes(this._scopes, length);
    }

    console.log("");

    console.log(
      `To get help for a specific task run: npx ${this._executableName} help [SCOPE] <TASK>\n`
    );
  }

  public printScopeHelp(
    scopeDefinition: ScopeDefinition,
    includeSubtasks = false
  ) {
    const name = scopeDefinition.name;
    const description = scopeDefinition.description ?? "";

    console.log(`${this._programName} version ${this._version}`);

    console.log(
      `\nUsage: hardhat [GLOBAL OPTIONS] ${name} <TASK> [TASK OPTIONS]`
    );

    console.log(`\nAVAILABLE TASKS:\n`);

    if (this._scopes[name] === undefined) {
      throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPE, {
        scope: name,
      });
    }

    this._printTasks(this._scopes[name].tasks, includeSubtasks);

    console.log(`\n${name}: ${description}`);

    console.log(
      `\nFor global options help run: ${this._executableName} help\n`
    );
  }

  public printTaskHelp(taskDefinition: TaskDefinition) {
    const {
      description = "",
      name,
      paramDefinitions,
      positionalParamDefinitions,
    } = taskDefinition;

    console.log(`${this._programName} version ${this._version}\n`);

    const paramsList = this._getParamsList(paramDefinitions);
    const positionalParamsList = this._getPositionalParamsList(
      positionalParamDefinitions
    );

    const scope =
      taskDefinition.scope !== undefined ? `${taskDefinition.scope} ` : "";

    console.log(
      `Usage: ${this._executableName} [GLOBAL OPTIONS] ${scope}${name}${paramsList}${positionalParamsList}\n`
    );

    if (Object.keys(paramDefinitions).length > 0) {
      console.log("OPTIONS:\n");

      this._printParamDetails(paramDefinitions);

      console.log("");
    }

    if (positionalParamDefinitions.length > 0) {
      console.log("POSITIONAL ARGUMENTS:\n");

      this._printPositionalParamDetails(positionalParamDefinitions);

      console.log("");
    }

    console.log(`${name}: ${description}\n`);

    console.log(`For global options help run: ${this._executableName} help\n`);
  }

  private _printTasks(
    tasksMap: TasksMap,
    includeSubtasks: boolean,
    length: number = 0
  ) {
    const taskNameList = Object.entries(tasksMap)
      .filter(
        ([, taskDefinition]) => includeSubtasks || !taskDefinition.isSubtask
      )
      .map(([taskName]) => taskName)
      .sort();

    const nameLength = taskNameList
      .map((n) => n.length)
      .reduce((a, b) => Math.max(a, b), length);

    for (const name of taskNameList) {
      const { description = "" } = tasksMap[name];

      console.log(`  ${name.padEnd(nameLength)}\t${description}`);
    }

    return nameLength;
  }

  private _printScopes(scopesMap: ScopesMap, length: number) {
    const scopeNamesList = Object.entries(scopesMap)
      .map(([scopeName]) => scopeName)
      .sort();

    const nameLength = scopeNamesList
      .map((n) => n.length)
      .reduce((a, b) => Math.max(a, b), length);

    for (const name of scopeNamesList) {
      const { description = "" } = scopesMap[name];

      console.log(`  ${name.padEnd(nameLength)}\t${description}`);
    }

    return nameLength;
  }

  private _getParamValueDescription<T>(paramDefinition: ParamDefinition<T>) {
    return `<${paramDefinition.type.name.toUpperCase()}>`;
  }

  private _getParamsList(paramDefinitions: ParamDefinitionsMap) {
    let paramsList = "";

    for (const name of Object.keys(paramDefinitions).sort()) {
      const definition = paramDefinitions[name];
      const { isFlag, isOptional } = definition;

      paramsList += " ";

      if (isOptional) {
        paramsList += "[";
      }

      paramsList += `${ArgumentsParser.paramNameToCLA(name)}`;

      if (!isFlag) {
        paramsList += ` ${this._getParamValueDescription(definition)}`;
      }

      if (isOptional) {
        paramsList += "]";
      }
    }

    return paramsList;
  }

  private _getPositionalParamsList(
    positionalParamDefinitions: Array<ParamDefinition<any>>
  ) {
    let paramsList = "";

    for (const definition of positionalParamDefinitions) {
      const { isOptional, isVariadic, name } = definition;

      paramsList += " ";

      if (isOptional) {
        paramsList += "[";
      }

      if (isVariadic) {
        paramsList += "...";
      }

      paramsList += name;

      if (isOptional) {
        paramsList += "]";
      }
    }

    return paramsList;
  }

  private _printParamDetails(paramDefinitions: ParamDefinitionsMap): number {
    const paramsNameLength = Object.keys(paramDefinitions)
      .map((n) => ArgumentsParser.paramNameToCLA(n).length)
      .reduce((a, b) => Math.max(a, b), 0);

    for (const name of Object.keys(paramDefinitions).sort()) {
      const { description, defaultValue, isOptional, isFlag } =
        paramDefinitions[name];

      let msg = `  ${ArgumentsParser.paramNameToCLA(name).padEnd(
        paramsNameLength
      )}\t`;

      if (description !== undefined) {
        msg += `${description} `;
      }

      if (isOptional && defaultValue !== undefined && !isFlag) {
        msg += `(default: ${JSON.stringify(defaultValue)})`;
      }

      console.log(msg);
    }

    return paramsNameLength;
  }

  private _printPositionalParamDetails(
    positionalParamDefinitions: Array<ParamDefinition<any>>
  ) {
    const paramsNameLength = positionalParamDefinitions
      .map((d) => d.name.length)
      .reduce((a, b) => Math.max(a, b), 0);

    for (const definition of positionalParamDefinitions) {
      const { name, description, isOptional, defaultValue } = definition;

      let msg = `  ${name.padEnd(paramsNameLength)}\t`;

      if (description !== undefined) {
        msg += `${description} `;
      }

      if (isOptional && defaultValue !== undefined) {
        msg += `(default: ${JSON.stringify(defaultValue)})`;
      }

      console.log(msg);
    }
  }
}
