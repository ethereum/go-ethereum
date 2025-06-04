export type ArgumentConfig<T> = {
    [P in keyof T]-?: PropertyConfig<T[P]>;
};

export type ArgumentOptions<T> = {
    [P in keyof T]-?: PropertyOptions<T[P]>;
};

interface OptionDefinition {
    name: string;
}

export type CommandLineOption<T = any> = PropertyOptions<T> & OptionDefinition;

export type PropertyConfig<T> = undefined extends T ? PropertyOptions<T> : RequiredPropertyOptions<T>;
export type RequiredPropertyOptions<T> = Array<any> extends T
    ? PropertyOptions<T>
    : TypeConstructor<T> | PropertyOptions<T>;

export type TypeConstructor<T> = (value: any) => T extends Array<infer R> ? R | undefined : T | undefined;

export type PropertyOptions<T> = {
    /**
     * A setter function (you receive the output from this) enabling you to be specific about the type and value received. Typical values
     * are `String`, `Number` and `Boolean` but you can use a custom function.
     */
    type: TypeConstructor<T>;

    /**
     * A getopt-style short option name. Can be any single character except a digit or hyphen.
     */
    alias?: string;

    /**
     * Set this flag if the option accepts multiple values. In the output, you will receive an array of values each passed through the `type` function.
     */
    multiple?: boolean;

    /**
     * Identical to `multiple` but with greedy parsing disabled.
     */
    lazyMultiple?: boolean;

    /**
     * Any values unaccounted for by an option definition will be set on the `defaultOption`. This flag is typically set
     * on the most commonly-used option to enable more concise usage.
     */
    defaultOption?: boolean;

    /**
     * An initial value for the option.
     */
    defaultValue?: T;

    /**
     * When your app has a large amount of options it makes sense to organise them in groups.
     *
     * There are two automatic groups: _all (contains all options) and _none (contains options without a group specified in their definition).
     */
    group?: string | string[];
    /** A string describing the option. */
    description?: string;
    /** A string to replace the default type string (e.g. <string>). It's often more useful to set a more descriptive type label, like <ms>, <files>, <command>, etc.. */
    typeLabel?: string;
} & OptionalPropertyOptions<T> &
    MultiplePropertyOptions<T>;

export type OptionalProperty = { optional: true };

export type OptionalPropertyOptions<T> = undefined extends T ? OptionalProperty : unknown;

export type MultiplePropertyOptions<T> = Array<any> extends T ? { multiple: true } | { lazyMultiple: true } : unknown;

export type HeaderLevel = 1 | 2 | 3 | 4 | 5;

export type PickType<T, TType> = Pick<T, { [K in keyof T]: Required<T>[K] extends TType ? K : never }[keyof T]>;

export interface UsageGuideOptions {
    /**
     * help sections to be listed before the options section
     */
    headerContentSections?: Content[];

    /**
     * help sections to be listed after the options section
     */
    footerContentSections?: Content[];

    /**
     * Used when generating error messages.
     * For example if a param is missing and there is a help option the error message will contain:
     *
     * 'To view help guide run myBaseCommand -h'
     */
    baseCommand?: string;

    /**
     * Heading level to use for the options header
     * Only used when generating markdown
     * Defaults to 2
     */
    optionsHeaderLevel?: HeaderLevel;

    /**
     * The header level to use for sections. Can be overridden on individual section definitions
     * defaults to 1
     */
    defaultSectionHeaderLevel?: HeaderLevel;

    /**
     * Heading level text to use for options section
     * defaults to "Options";
     */
    optionsHeaderText?: string;

    /**
     * Used to define multiple options sections. If this is used `optionsHeaderLevel` and `optionsHeaderText` are ignored.
     */
    optionSections?: OptionContent[];
}

export interface ArgsParseOptions<T> extends UsageGuideOptions {
    /**
     * An array of strings which if present will be parsed instead of `process.argv`.
     */
    argv?: string[];

    /**
     * A logger for printing errors for missing properties.
     * Defaults to console
     */
    logger?: typeof console;

    /**
     * The command line argument used to show help
     * By default when this property is true help will be printed and the process will exit
     */
    helpArg?: keyof PickType<T, boolean>;

    /**
     * The command line argument with path of file to load arguments from
     * If this property is set the file will be loaded and used to create the returned arguments object.
     * The file can contain a partial object, missing required arguments must be specified on the command line
     * Any arguments specified on the command line will override those specified in the file.
     * The config object must be all strings (or arrays of strings) that will then be passed to the type function specified for that argument
     * For boolean use:
     * {
     *  myBooleanArg: "true"
     * }
     */
    loadFromFileArg?: keyof PickType<T, string>;

    /**
     * The command line argument specifying the json path of the config object within the file
     * If loadFromFileArg is specified the json path is used to locate the config object in the loaded json file
     * If not specified the whole file will be used
     * This allows the specification to be defined within the package.json file for example:
     * loadFromFileJsonPath: "config.writeMarkdown"
     * package.json:
     * {
     *  name: "myApp",
     *  version: "1.1.1",
     *  dependencies: {},
     *  config: {
     *      writeMarkdown: {
     *          markdownPath: [ "myMarkdownFile.md" ]
     *      }
     *  }
     * }
     */
    loadFromFileJsonPathArg?: keyof PickType<T, string>;

    /**
     * When set to true the error message stating which arguments are missing are not printed
     */
    hideMissingArgMessages?: boolean;

    /**
     * By default when a required arg is missing an error will be thrown.
     * If this set to true the usage guide will be printed out instead
     */
    showHelpWhenArgsMissing?: boolean;

    /**
     * If showHelpWhenArgsMissing is enabled this header section is displayed before the help content.
     * A static section can be defined or a function that will return a section. This function is passed an array of required params that where not supplied.
     */
    helpWhenArgMissingHeader?:
        | ((missingArgs: CommandLineOption[]) => Omit<Content, 'includeIn'>)
        | Omit<Content, 'includeIn'>;

    /**
     * adds a (O), (D) or both to typeLabel to indicate if a property is optional or the default option
     */
    displayOptionalAndDefault?: boolean;

    /**
     * if displayOptionalAndDefault is true and any params are optional or default adds a footer explaining what the (O), (D) means
     */
    addOptionalDefaultExplanatoryFooter?: boolean;

    /**
     * prepends the supplied description with details about the param. These include default option, optional and the default value.
     */
    prependParamOptionsToDescription?: boolean;

    /**
     * sets the exit code of the process when exiting early due to missing args or showing usage guide
     * 0 will be used for an exit code if this is not specified.
     */
    processExitCode?: number | ProcessExitCodeFunction<T>;
}

export type ProcessExitCodeFunction<T> = (
    reason: ExitReason,
    passedArgs: Partial<T>,
    missingArgs: CommandLineOption<any>[],
) => number;
export type ExitReason = 'missingArgs' | 'usageGuide';

export interface PartialParseOptions extends ArgsParseOptions<any> {
    /**
     * If `true`, `commandLineArgs` will not throw on unknown options or values, instead returning them in the `_unknown` property of the output.
     */
    partial: true;
}

export interface StopParseOptions extends ArgsParseOptions<any> {
    /**
     * If `true`, `commandLineArgs` will not throw on unknown options or values. Instead, parsing will stop at the first unknown argument
     * and the remaining arguments returned in the `_unknown` property of the output. If set, `partial: true` is implied.
     */
    stopAtFirstUnknown: true;
}

export type CommandLineResults<R extends boolean> = R extends false
    ? // eslint-disable-next-line @typescript-eslint/ban-types
      {}
    : {
          _commandLineResults: {
              missingArgs: CommandLineOption[];
              printHelp: () => void;
          };
      };

type UnknownProps = { _unknown: string[] };

export type UnknownProperties<T> = T extends PartialParseOptions
    ? UnknownProps
    : T extends StopParseOptions
    ? UnknownProps
    : unknown;

export type ParseOptions<T> = ArgsParseOptions<T> | PartialParseOptions | StopParseOptions;

export interface SectionHeader {
    /** The section header, always bold and underlined. */
    header?: string;

    /**
     * Heading level to use for the header
     * Only used when generating markdown
     * Defaults to 1
     */
    headerLevel?: HeaderLevel;
}

export interface OptionContent extends SectionHeader {
    /** The group name or names. use '_none' for options without a group */
    group?: string | string[];
    /** The names of one of more option definitions to hide from the option list.  */
    hide?: string | string[];
    /** If true, the option alias will be displayed after the name, i.e. --verbose, -v instead of -v, --verbose). */
    reverseNameOrder?: boolean;
}

/** A Content section comprises a header and one or more lines of content. */
export interface Content extends SectionHeader {
    /**
     * Overloaded property, accepting data in one of four formats.
     *  1. A single string (one line of text).
     *  2. An array of strings (multiple lines of text).
     *  3. An array of objects (recordset-style data). In this case, the data will be rendered in table format. The property names of each object are not important, so long as they are
     *     consistent throughout the array.
     *  4. An object with two properties - data and options. In this case, the data and options will be passed directly to the underlying table layout module for rendering.
     */
    content?: string | string[] | any[];

    includeIn?: 'markdown' | 'cli' | 'both';
}

export interface IInsertCodeOptions {
    insertCodeBelow?: string;
    insertCodeAbove?: string;
    copyCodeBelow?: string;
    copyCodeAbove?: string;
    removeDoubleBlankLines: boolean;
}

export interface IReplaceOptions {
    replaceBelow?: string;
    replaceAbove?: string;
    removeDoubleBlankLines: boolean;
}

export type JsImport = { jsFile: string; importName: string };

export interface IWriteMarkDown extends IReplaceOptions, IInsertCodeOptions {
    markdownPath: string;
    jsFile?: string[];
    configImportName: string[];
    help: boolean;
    verify: boolean;
    configFile?: string;
    jsonPath?: string;
    verifyMessage?: string;
    skipFooter: boolean;
}

export type UsageGuideConfig<T = any> = {
    arguments: ArgumentConfig<T>;
    parseOptions?: ParseOptions<T>;
};

export interface OptionList {
    header?: string;
    /** An array of option definition objects. */
    optionList?: OptionDefinition[];
    /** If specified, only options from this particular group will be printed.  */
    group?: string | string[];
    /** The names of one of more option definitions to hide from the option list.  */
    hide?: string | string[];
    /** If true, the option alias will be displayed after the name, i.e. --verbose, -v instead of -v, --verbose). */
    reverseNameOrder?: boolean;
    /** An options object suitable for passing into table-layout. */
    tableOptions?: any;
}
