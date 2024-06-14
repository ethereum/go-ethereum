import ProcessEnv = NodeJS.ProcessEnv;

import { HardhatArguments, HardhatParamDefinitions } from "../../../types";
import { ArgumentsParser } from "../../cli/ArgumentsParser";
import { unsafeObjectKeys } from "../../util/unsafe";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";

const HARDHAT_ENV_ARGUMENT_PREFIX = "HARDHAT_";

export function paramNameToEnvVariable(paramName: string): string {
  // We create it starting from the result of ArgumentsParser.paramNameToCLA
  // so it's easier to explain and understand their equivalences.
  return ArgumentsParser.paramNameToCLA(paramName)
    .replace(ArgumentsParser.PARAM_PREFIX, HARDHAT_ENV_ARGUMENT_PREFIX)
    .replace(/-/g, "_")
    .toUpperCase();
}

export function getEnvVariablesMap(hardhatArguments: HardhatArguments): {
  [envVar: string]: string;
} {
  const values: { [envVar: string]: string } = {};

  for (const [name, value] of Object.entries(hardhatArguments)) {
    if (value === undefined) {
      continue;
    }

    values[paramNameToEnvVariable(name)] = value.toString();
  }

  return values;
}

export function getEnvHardhatArguments(
  paramDefinitions: HardhatParamDefinitions,
  envVariables: ProcessEnv
): HardhatArguments {
  const envArgs: any = {};

  for (const paramName of unsafeObjectKeys(paramDefinitions)) {
    const definition = paramDefinitions[paramName];
    const envVarName = paramNameToEnvVariable(paramName);
    const rawValue = envVariables[envVarName];

    if (rawValue !== undefined) {
      try {
        envArgs[paramName] = definition.type.parse(paramName, rawValue);
      } catch (error) {
        if (error instanceof Error) {
          throw new HardhatError(
            ERRORS.ARGUMENTS.INVALID_ENV_VAR_VALUE,
            {
              varName: envVarName,
              value: rawValue,
            },
            error
          );
        }

        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
      }
    } else {
      envArgs[paramName] = definition.defaultValue;
    }
  }

  // TODO: This is a little type-unsafe, but we know we have all the needed arguments
  return envArgs as HardhatArguments;
}
