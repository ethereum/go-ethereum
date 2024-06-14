import { ArgumentConfig, ArgumentOptions, CommandLineOption } from '../contracts';
export declare function createCommandLineConfig<T>(config: ArgumentOptions<T>): CommandLineOption[];
export declare function normaliseConfig<T>(config: ArgumentConfig<T>): ArgumentOptions<T>;
export declare function mergeConfig<T>(parsedConfig: Partial<T>, parsedConfigWithoutDefaults: Partial<T>, fileContent: Record<string, unknown>, options: ArgumentOptions<T>, jsonPath: keyof T | undefined): Partial<T>;
/**
 * commandLineArgs throws an error if we pass aa value for a boolean arg as follows:
 * myCommand -a=true --booleanArg=false --otherArg true
 * this function removes these booleans so as to avoid errors from commandLineArgs
 * @param args
 * @param config
 */
export declare function removeBooleanValues<T>(args: string[], config: ArgumentOptions<T>): string[];
/**
 * Gets the values of any boolean arguments that were specified on the command line with a value
 * These arguments were removed by removeBooleanValues
 * @param args
 * @param config
 */
export declare function getBooleanValues<T>(args: string[], config: ArgumentOptions<T>): Partial<T>;
