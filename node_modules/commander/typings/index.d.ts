// Type definitions for commander 2.11
// Project: https://github.com/visionmedia/commander.js
// Definitions by: Alan Agius <https://github.com/alan-agius4>, Marcelo Dezem <https://github.com/mdezem>, vvakame <https://github.com/vvakame>, Jules Randolph <https://github.com/sveinburne>
// Definitions: https://github.com/DefinitelyTyped/DefinitelyTyped

///<reference types="node" />

declare namespace local {

  class Option {
    flags: string;
    required: boolean;
    optional: boolean;
    bool: boolean;
    short?: string;
    long: string;
    description: string;

    /**
     * Initialize a new `Option` with the given `flags` and `description`.
     *
     * @param {string} flags
     * @param {string} [description]
     */
    constructor(flags: string, description?: string);
  }

  class Command extends NodeJS.EventEmitter {
    [key: string]: any;

    args: string[];

    /**
     * Initialize a new `Command`.
     *
     * @param {string} [name]
     */
    constructor(name?: string);

    /**
     * Set the program version to `str`. 
     *
     * This method auto-registers the "-V, --version" flag
     * which will print the version number when passed.
     * 
     * You can optionally supply the  flags and description to override the defaults.
     *
     */
    version(str: string, flags?: string, description?: string): Command;

    /**
     * Define a command, implemented using an action handler.
     * 
     * @remarks
     * The command description is supplied using `.description`, not as a parameter to `.command`.
     * 
     * @example
     * ```ts
     *  program
     *    .command('clone <source> [destination]')
     *    .description('clone a repository into a newly created directory')
     *    .action((source, destination) => {
     *      console.log('clone command called');
     *    });
     * ```
     * 
     * @param nameAndArgs - command name and arguments, args are  `<required>` or `[optional]` and last may also be `variadic...`
     * @param opts - configuration options
     * @returns new command
     */
    command(nameAndArgs: string, opts?: commander.CommandOptions): Command;
    /**
     * Define a command, implemented in a separate executable file.
     * 
     * @remarks
     * The command description is supplied as the second parameter to `.command`.
     * 
     * @example
     * ```ts
     *  program
     *    .command('start <service>', 'start named service')
     *    .command('stop [service]', 'stop named serice, or all if no name supplied');
     * ```
     * 
     * @param nameAndArgs - command name and arguments, args are  `<required>` or `[optional]` and last may also be `variadic...`
     * @param description - description of executable command
     * @param opts - configuration options
     * @returns top level command for chaining more command definitions
     */
    command(nameAndArgs: string, description: string, opts?: commander.CommandOptions): Command;

    /**
     * Define argument syntax for the top-level command.
     *
     * @param {string} desc
     * @returns {Command} for chaining
     */
    arguments(desc: string): Command;

    /**
     * Parse expected `args`.
     *
     * For example `["[type]"]` becomes `[{ required: false, name: 'type' }]`.
     *
     * @param {string[]} args
     * @returns {Command} for chaining
     */
    parseExpectedArgs(args: string[]): Command;

    /**
     * Register callback `fn` for the command.
     *
     * @example
     *      program
     *        .command('help')
     *        .description('display verbose help')
     *        .action(function() {
     *           // output help here
     *        });
     *
     * @param {(...args: any[]) => void} fn
     * @returns {Command} for chaining
     */
    action(fn: (...args: any[]) => void): Command;

    /**
     * Define option with `flags`, `description` and optional
     * coercion `fn`.
     *
     * The `flags` string should contain both the short and long flags,
     * separated by comma, a pipe or space. The following are all valid
     * all will output this way when `--help` is used.
     *
     *    "-p, --pepper"
     *    "-p|--pepper"
     *    "-p --pepper"
     *
     * @example
     *     // simple boolean defaulting to false
     *     program.option('-p, --pepper', 'add pepper');
     *
     *     --pepper
     *     program.pepper
     *     // => Boolean
     *
     *     // simple boolean defaulting to true
     *     program.option('-C, --no-cheese', 'remove cheese');
     *
     *     program.cheese
     *     // => true
     *
     *     --no-cheese
     *     program.cheese
     *     // => false
     *
     *     // required argument
     *     program.option('-C, --chdir <path>', 'change the working directory');
     *
     *     --chdir /tmp
     *     program.chdir
     *     // => "/tmp"
     *
     *     // optional argument
     *     program.option('-c, --cheese [type]', 'add cheese [marble]');
     *
     * @param {string} flags
     * @param {string} [description]
     * @param {((arg1: any, arg2: any) => void) | RegExp} [fn] function or default
     * @param {*} [defaultValue]
     * @returns {Command} for chaining
     */
    option(flags: string, description?: string, fn?: ((arg1: any, arg2: any) => void) | RegExp, defaultValue?: any): Command;
    option(flags: string, description?: string, defaultValue?: any): Command;

    /**
     * Allow unknown options on the command line.
     *
     * @param {boolean} [arg] if `true` or omitted, no error will be thrown for unknown options.
     * @returns {Command} for chaining
     */
    allowUnknownOption(arg?: boolean): Command;

    /**
     * Parse `argv`, settings options and invoking commands when defined.
     *
     * @param {string[]} argv
     * @returns {Command} for chaining
     */
    parse(argv: string[]): Command;

    /**
     * Parse options from `argv` returning `argv` void of these options.
     *
     * @param {string[]} argv
     * @returns {ParseOptionsResult}
     */
    parseOptions(argv: string[]): commander.ParseOptionsResult;

    /**
     * Return an object containing options as key-value pairs
     *
     * @returns {{[key: string]: any}}
     */
    opts(): { [key: string]: any };

    /**
     * Set the description to `str`.
     *
     * @param {string} str
     * @param {{[argName: string]: string}} argsDescription
     * @return {(Command | string)}
     */
    description(str: string, argsDescription?: {[argName: string]: string}): Command;
    description(): string;

    /**
     * Set an alias for the command.
     *
     * @param {string} alias
     * @return {(Command | string)}
     */
    alias(alias: string): Command;
    alias(): string;

    /**
     * Set or get the command usage.
     *
     * @param {string} str
     * @return {(Command | string)}
     */
    usage(str: string): Command;
    usage(): string;

    /**
     * Set the name of the command.
     *
     * @param {string} str
     * @return {Command}
     */
    name(str: string): Command;

    /**
     * Get the name of the command.
     *
     * @return {string}
     */
    name(): string;

    /**
     * Output help information for this command.
     *
     * When listener(s) are available for the helpLongFlag
     * those callbacks are invoked.
     * 
     * @param {(str: string) => string} [cb]
     */
    outputHelp(cb?: (str: string) => string): void;

    /**
     * You can pass in flags and a description to override the help
     * flags and help description for your command.
     */
    helpOption(flags?: string, description?: string): Command;

    /** 
     * Output help information and exit.
     */
    help(cb?: (str: string) => string): never;
  }

}

declare namespace commander {

    type Command = local.Command

    type Option = local.Option

    interface CommandOptions {
        noHelp?: boolean;
        isDefault?: boolean;
        executableFile?: string;
    }

    interface ParseOptionsResult {
        args: string[];
        unknown: string[];
    }

    interface CommanderStatic extends Command {
        Command: typeof local.Command;
        Option: typeof local.Option;
        CommandOptions: CommandOptions;
        ParseOptionsResult: ParseOptionsResult;
    }

}

declare const commander: commander.CommanderStatic;
export = commander;
