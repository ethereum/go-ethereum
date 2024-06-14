import * as fs from "fs";
import fsExtra from "fs-extra";

import { ArgumentType, CLIArgumentType } from "../../../types";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";

/**
 * String type.
 *
 * Accepts any kind of string.
 */
export const string: CLIArgumentType<string> = {
  name: "string",
  parse: (argName, strValue) => strValue,
  /**
   * Check if argument value is of type "string"
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "string"
   */
  validate: (argName: string, value: any): void => {
    const isString = typeof value === "string";

    if (!isString) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: string.name,
      });
    }
  },
};

/**
 * Boolean type.
 *
 * Accepts only 'true' or 'false' (case-insensitive).
 * @throws HH301
 */
export const boolean: CLIArgumentType<boolean> = {
  name: "boolean",
  parse: (argName, strValue) => {
    if (strValue.toLowerCase() === "true") {
      return true;
    }
    if (strValue.toLowerCase() === "false") {
      return false;
    }

    throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
      value: strValue,
      name: argName,
      type: "boolean",
    });
  },
  /**
   * Check if argument value is of type "boolean"
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "boolean"
   */
  validate: (argName: string, value: any): void => {
    const isBoolean = typeof value === "boolean";

    if (!isBoolean) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: boolean.name,
      });
    }
  },
};

/**
 * Int type.
 * Accepts either a decimal string integer or hexadecimal string integer.
 * @throws HH301
 */
export const int: CLIArgumentType<number> = {
  name: "int",
  parse: (argName, strValue) => {
    const decimalPattern = /^\d+(?:[eE]\d+)?$/;
    const hexPattern = /^0[xX][\dABCDEabcde]+$/;

    if (
      strValue.match(decimalPattern) === null &&
      strValue.match(hexPattern) === null
    ) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value: strValue,
        name: argName,
        type: int.name,
      });
    }

    return Number(strValue);
  },
  /**
   * Check if argument value is of type "int"
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "int"
   */
  validate: (argName: string, value: any): void => {
    const isInt = Number.isInteger(value);
    if (!isInt) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: int.name,
      });
    }
  },
};

/**
 * BigInt type.
 * Accepts either a decimal string integer or hexadecimal string integer.
 * @throws HH301
 */
export const bigint: CLIArgumentType<bigint> = {
  name: "bigint",
  parse: (argName, strValue) => {
    const decimalPattern = /^\d+(?:n)?$/;
    const hexPattern = /^0[xX][\dABCDEabcde]+$/;

    if (
      strValue.match(decimalPattern) === null &&
      strValue.match(hexPattern) === null
    ) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value: strValue,
        name: argName,
        type: bigint.name,
      });
    }

    return BigInt(strValue.replace("n", ""));
  },
  /**
   * Check if argument value is of type "bigint".
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "bigint"
   */
  validate: (argName: string, value: any): void => {
    const isBigInt = typeof value === "bigint";
    if (!isBigInt) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: bigint.name,
      });
    }
  },
};

/**
 * Float type.
 * Accepts either a decimal string number or hexadecimal string number.
 * @throws HH301
 */
export const float: CLIArgumentType<number> = {
  name: "float",
  parse: (argName, strValue) => {
    const decimalPattern = /^(?:\d+(?:\.\d*)?|\.\d+)(?:[eE]\d+)?$/;
    const hexPattern = /^0[xX][\dABCDEabcde]+$/;

    if (
      strValue.match(decimalPattern) === null &&
      strValue.match(hexPattern) === null
    ) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value: strValue,
        name: argName,
        type: float.name,
      });
    }

    return Number(strValue);
  },
  /**
   * Check if argument value is of type "float".
   * Both decimal and integer number values are valid.
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "number"
   */
  validate: (argName: string, value: any): void => {
    const isFloatOrInteger = typeof value === "number" && !isNaN(value);

    if (!isFloatOrInteger) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: float.name,
      });
    }
  },
};

/**
 * Input file type.
 * Accepts a path to a readable file..
 * @throws HH302
 */
export const inputFile: CLIArgumentType<string> = {
  name: "inputFile",
  parse(argName: string, strValue: string): string {
    try {
      fs.accessSync(strValue, fsExtra.constants.R_OK);
      const stats = fs.lstatSync(strValue);

      if (stats.isDirectory()) {
        // This is caught and encapsulated in a hardhat error.
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new Error(`${strValue} is a directory, not a file`);
      }
    } catch (error) {
      if (error instanceof Error) {
        throw new HardhatError(
          ERRORS.ARGUMENTS.INVALID_INPUT_FILE,
          {
            name: argName,
            value: strValue,
          },
          error
        );
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }

    return strValue;
  },

  /**
   * Check if argument value is of type "inputFile"
   * File string validation succeeds if it can be parsed, ie. is a valid accessible file dir
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "inputFile"
   */
  validate: (argName: string, value: any): void => {
    try {
      inputFile.parse(argName, value);
    } catch (error) {
      // the input value is considered invalid, throw error.
      if (error instanceof Error) {
        throw new HardhatError(
          ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE,
          {
            value,
            name: argName,
            type: inputFile.name,
          },
          error
        );
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  },
};

export const json: CLIArgumentType<any> = {
  name: "json",
  parse(argName: string, strValue: string): any {
    try {
      return JSON.parse(strValue);
    } catch (error) {
      if (error instanceof Error) {
        throw new HardhatError(
          ERRORS.ARGUMENTS.INVALID_JSON_ARGUMENT,
          {
            param: argName,
            error: error.message,
          },
          error
        );
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  },

  /**
   * Check if argument value is of type "json". We consider everything except
   * undefined to be json.
   *
   * @param argName {string} argument's name - used for context in case of error.
   * @param value {any} argument's value to validate.
   *
   * @throws HH301 if value is not of type "json"
   */
  validate: (argName: string, value: any): void => {
    if (value === undefined) {
      throw new HardhatError(ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
        value,
        name: argName,
        type: json.name,
      });
    }
  },
};

export const any: ArgumentType<any> = {
  name: "any",
  validate(_argName: string, _argumentValue: any) {},
};
