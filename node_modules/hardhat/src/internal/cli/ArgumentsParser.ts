import { TASK_HELP } from "../../builtin-tasks/task-names";
import {
  CLIArgumentType,
  HardhatArguments,
  HardhatParamDefinitions,
  ParamDefinition,
  ParamDefinitionsMap,
  ScopesMap,
  TaskArguments,
  TaskDefinition,
  TasksMap,
} from "../../types";
import { HardhatError } from "../core/errors";
import { ERRORS } from "../core/errors-list";

export class ArgumentsParser {
  public static readonly PARAM_PREFIX = "--";

  public static paramNameToCLA(paramName: string): string {
    return (
      ArgumentsParser.PARAM_PREFIX +
      paramName
        .split(/(?=[A-Z])/g)
        .map((s) => s.toLowerCase())
        .join("-")
    );
  }

  public static cLAToParamName(cLA: string): string {
    if (cLA.toLowerCase() !== cLA) {
      throw new HardhatError(ERRORS.ARGUMENTS.PARAM_NAME_INVALID_CASING, {
        param: cLA,
      });
    }

    const parts = cLA
      .slice(ArgumentsParser.PARAM_PREFIX.length)
      .split("-")
      .filter((x) => x.length > 0);

    return (
      parts[0] +
      parts
        .slice(1)
        .map((s) => s[0].toUpperCase() + s.slice(1))
        .join("")
    );
  }

  public parseHardhatArguments(
    hardhatParamDefinitions: HardhatParamDefinitions,
    envVariableArguments: HardhatArguments,
    rawCLAs: string[]
  ): {
    hardhatArguments: HardhatArguments;
    scopeOrTaskName: string | undefined;
    allUnparsedCLAs: string[];
  } {
    const hardhatArguments: Partial<HardhatArguments> = {};
    let scopeOrTaskName: string | undefined;
    const allUnparsedCLAs: string[] = [];

    for (let i = 0; i < rawCLAs.length; i++) {
      const arg = rawCLAs[i];

      if (scopeOrTaskName === undefined) {
        if (!this._hasCLAParamNameFormat(arg)) {
          scopeOrTaskName = arg;
          allUnparsedCLAs.push(arg);
          continue;
        }

        if (!this._isCLAParamName(arg, hardhatParamDefinitions)) {
          throw new HardhatError(
            ERRORS.ARGUMENTS.UNRECOGNIZED_COMMAND_LINE_ARG,
            { argument: arg }
          );
        }

        i = this._parseArgumentAt(
          rawCLAs,
          i,
          hardhatParamDefinitions,
          hardhatArguments,
          scopeOrTaskName
        );
      } else {
        if (!this._isCLAParamName(arg, hardhatParamDefinitions)) {
          allUnparsedCLAs.push(arg);
          continue;
        }

        i = this._parseArgumentAt(
          rawCLAs,
          i,
          hardhatParamDefinitions,
          hardhatArguments,
          scopeOrTaskName
        );
      }
    }

    return {
      hardhatArguments: this._addHardhatDefaultArguments(
        hardhatParamDefinitions,
        envVariableArguments,
        hardhatArguments
      ),
      scopeOrTaskName,
      allUnparsedCLAs,
    };
  }

  public parseScopeAndTaskNames(
    allUnparsedCLAs: string[],
    taskDefinitions: TasksMap,
    scopeDefinitions: ScopesMap
  ): {
    scopeName?: string;
    taskName: string;
    unparsedCLAs: string[];
  } {
    const [firstCLA, secondCLA] = allUnparsedCLAs;

    if (allUnparsedCLAs.length === 0) {
      return {
        taskName: TASK_HELP,
        unparsedCLAs: [],
      };
    } else if (allUnparsedCLAs.length === 1) {
      if (scopeDefinitions[firstCLA] !== undefined) {
        // this is a bit of a hack, but it's the easiest way to print
        // the help of a scope when no task is specified
        return {
          taskName: TASK_HELP,
          unparsedCLAs: [firstCLA],
        };
      } else if (taskDefinitions[firstCLA] !== undefined) {
        return {
          taskName: firstCLA,
          unparsedCLAs: allUnparsedCLAs.slice(1),
        };
      } else {
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
          task: firstCLA,
        });
      }
    } else {
      const scopeDefinition = scopeDefinitions[firstCLA];
      if (scopeDefinition !== undefined) {
        if (scopeDefinition.tasks[secondCLA] !== undefined) {
          return {
            scopeName: firstCLA,
            taskName: secondCLA,
            unparsedCLAs: allUnparsedCLAs.slice(2),
          };
        } else {
          throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
            scope: firstCLA,
            task: secondCLA,
          });
        }
      } else if (taskDefinitions[firstCLA] !== undefined) {
        return {
          taskName: firstCLA,
          unparsedCLAs: allUnparsedCLAs.slice(1),
        };
      } else {
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
          task: firstCLA,
        });
      }
    }
  }

  public parseTaskArguments(
    taskDefinition: TaskDefinition,
    rawCLAs: string[]
  ): TaskArguments {
    const { paramArguments, rawPositionalArguments } =
      this._parseTaskParamArguments(taskDefinition, rawCLAs);

    const positionalArguments = this._parsePositionalParamArgs(
      rawPositionalArguments,
      taskDefinition.positionalParamDefinitions
    );

    return { ...paramArguments, ...positionalArguments };
  }

  private _parseTaskParamArguments(
    taskDefinition: TaskDefinition,
    rawCLAs: string[]
  ) {
    const paramArguments = {};
    const rawPositionalArguments: string[] = [];

    for (let i = 0; i < rawCLAs.length; i++) {
      const arg = rawCLAs[i];

      if (!this._hasCLAParamNameFormat(arg)) {
        rawPositionalArguments.push(arg);
        continue;
      }

      if (!this._isCLAParamName(arg, taskDefinition.paramDefinitions)) {
        throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_PARAM_NAME, {
          param: arg,
        });
      }

      i = this._parseArgumentAt(
        rawCLAs,
        i,
        taskDefinition.paramDefinitions,
        paramArguments,
        taskDefinition.name
      );
    }

    this._addTaskDefaultArguments(taskDefinition, paramArguments);

    return { paramArguments, rawPositionalArguments };
  }

  private _addHardhatDefaultArguments(
    hardhatParamDefinitions: HardhatParamDefinitions,
    envVariableArguments: HardhatArguments,
    hardhatArguments: Partial<HardhatArguments>
  ): HardhatArguments {
    return {
      ...envVariableArguments,
      ...hardhatArguments,
    };
  }

  private _addTaskDefaultArguments(
    taskDefinition: TaskDefinition,
    taskArguments: TaskArguments
  ) {
    for (const paramName of Object.keys(taskDefinition.paramDefinitions)) {
      const definition = taskDefinition.paramDefinitions[paramName];

      if (taskArguments[paramName] !== undefined) {
        continue;
      }
      if (!definition.isOptional) {
        throw new HardhatError(ERRORS.ARGUMENTS.MISSING_TASK_ARGUMENT, {
          param: ArgumentsParser.paramNameToCLA(paramName),
          task: taskDefinition.name,
        });
      }

      taskArguments[paramName] = definition.defaultValue;
    }
  }

  private _isCLAParamName(str: string, paramDefinitions: ParamDefinitionsMap) {
    if (!this._hasCLAParamNameFormat(str)) {
      return false;
    }

    const name = ArgumentsParser.cLAToParamName(str);
    return paramDefinitions[name] !== undefined;
  }

  private _hasCLAParamNameFormat(str: string) {
    return str.startsWith(ArgumentsParser.PARAM_PREFIX);
  }

  private _parseArgumentAt(
    rawCLAs: string[],
    index: number,
    paramDefinitions: ParamDefinitionsMap,
    parsedArguments: TaskArguments,
    scopeOrTaskName?: string
  ) {
    const claArg = rawCLAs[index];
    const paramName = ArgumentsParser.cLAToParamName(claArg);
    const definition = paramDefinitions[paramName];

    if (parsedArguments[paramName] !== undefined) {
      throw new HardhatError(ERRORS.ARGUMENTS.REPEATED_PARAM, {
        param: claArg,
      });
    }

    if (definition.isFlag) {
      parsedArguments[paramName] = true;
    } else {
      index++;
      const value = rawCLAs[index];

      if (value === undefined) {
        throw new HardhatError(ERRORS.ARGUMENTS.MISSING_TASK_ARGUMENT, {
          param: ArgumentsParser.paramNameToCLA(paramName),
          task: scopeOrTaskName ?? "help",
        });
      }

      // We only parse the arguments of non-subtasks, and those only
      // accept CLIArgumentTypes.
      const type = definition.type as CLIArgumentType<any>;
      parsedArguments[paramName] = type.parse(paramName, value);
    }

    return index;
  }

  private _parsePositionalParamArgs(
    rawPositionalParamArgs: string[],
    positionalParamDefinitions: Array<ParamDefinition<any>>
  ): TaskArguments {
    const args: TaskArguments = {};

    for (let i = 0; i < positionalParamDefinitions.length; i++) {
      const definition = positionalParamDefinitions[i];
      // We only parse the arguments of non-subtasks, and those only
      // accept CLIArgumentTypes.
      const type = definition.type as CLIArgumentType<any>;

      const rawArg = rawPositionalParamArgs[i];

      if (rawArg === undefined) {
        if (!definition.isOptional) {
          throw new HardhatError(ERRORS.ARGUMENTS.MISSING_POSITIONAL_ARG, {
            param: definition.name,
          });
        }

        args[definition.name] = definition.defaultValue;
      } else if (!definition.isVariadic) {
        args[definition.name] = type.parse(definition.name, rawArg);
      } else {
        args[definition.name] = rawPositionalParamArgs
          .slice(i)
          .map((raw) => type.parse(definition.name, raw));
      }
    }

    const lastDefinition =
      positionalParamDefinitions[positionalParamDefinitions.length - 1];

    const hasVariadicParam =
      lastDefinition !== undefined && lastDefinition.isVariadic;

    if (
      !hasVariadicParam &&
      rawPositionalParamArgs.length > positionalParamDefinitions.length
    ) {
      throw new HardhatError(ERRORS.ARGUMENTS.UNRECOGNIZED_POSITIONAL_ARG, {
        argument: rawPositionalParamArgs[positionalParamDefinitions.length],
      });
    }

    return args;
  }
}
