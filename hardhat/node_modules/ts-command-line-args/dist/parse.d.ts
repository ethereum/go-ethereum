import { ArgumentConfig, ParseOptions, UnknownProperties, CommandLineResults } from './contracts';
/**
 * parses command line arguments and returns an object with all the arguments in IF all required options passed
 * @param config the argument config. Required, used to determine what arguments are expected
 * @param options
 * @param exitProcess defaults to true. The process will exit if any required arguments are omitted
 * @param addCommandLineResults defaults to false. If passed an additional _commandLineResults object will be returned in the result
 * @returns
 */
export declare function parse<T, P extends ParseOptions<T> = ParseOptions<T>, R extends boolean = false>(config: ArgumentConfig<T>, options?: P, exitProcess?: boolean, addCommandLineResults?: R): T & UnknownProperties<P> & CommandLineResults<R>;
