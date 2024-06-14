import {
    UsageGuideConfig,
    JsImport,
    IWriteMarkDown,
    ArgumentConfig,
    CommandLineOption,
    ParseOptions,
    Content,
    HeaderLevel,
    OptionContent,
    SectionHeader,
} from '../contracts';
import { join } from 'path';
import { normaliseConfig, createCommandLineConfig } from './command-line.helper';
import { generateTableFooter, getOptionSections, mapDefinitionDetails } from './options.helper';
import { convertChalkStringToMarkdown } from './string.helper';

export function createUsageGuide<T = any>(config: UsageGuideConfig<T>): string {
    const options = config.parseOptions || {};
    const headerSections = options.headerContentSections || [];
    const footerSections = options.footerContentSections || [];

    return [
        ...headerSections.filter(filterMarkdownSections).map((section) => createSection(section, config)),
        ...createOptionsSections(config.arguments, options),
        ...footerSections.filter(filterMarkdownSections).map((section) => createSection(section, config)),
    ].join('\n');
}

function filterMarkdownSections(section: Content): boolean {
    return section.includeIn == null || section.includeIn === 'both' || section.includeIn === 'markdown';
}

export function createSection<T>(section: Content, config: UsageGuideConfig<T>): string {
    return `
${createHeading(section, config.parseOptions?.defaultSectionHeaderLevel || 1)}
${createSectionContent(section)}
`;
}

export function createSectionContent(section: Content): string {
    if (typeof section.content === 'string') {
        return convertChalkStringToMarkdown(section.content);
    }

    if (Array.isArray(section.content)) {
        if (section.content.every((content) => typeof content === 'string')) {
            return (section.content as string[]).map(convertChalkStringToMarkdown).join('\n');
        } else if (section.content.every((content) => typeof content === 'object')) {
            return createSectionTable(section.content);
        }
    }

    return '';
}

export function createSectionTable(rows: any[]): string {
    if (rows.length === 0) {
        return ``;
    }
    const cellKeys = Object.keys(rows[0]);

    return `
|${cellKeys.map((key) => ` ${key} `).join('|')}|
|${cellKeys.map(() => '-').join('|')}|
${rows.map((row) => `| ${cellKeys.map((key) => convertChalkStringToMarkdown(row[key])).join(' | ')} |`).join('\n')}`;
}

export function createOptionsSections<T>(cliArguments: ArgumentConfig<T>, options: ParseOptions<any>): string[] {
    const normalisedConfig = normaliseConfig(cliArguments);
    const optionList = createCommandLineConfig(normalisedConfig);

    if (optionList.length === 0) {
        return [];
    }

    return getOptionSections(options).map((section) => createOptionsSection(optionList, section, options));
}

export function createOptionsSection(
    optionList: CommandLineOption<any>[],
    content: OptionContent,
    options: ParseOptions<any>,
): string {
    optionList = optionList.filter((option) => filterOptions(option, content.group));
    const anyAlias = optionList.some((option) => option.alias != null);
    const anyDescription = optionList.some((option) => option.description != null);

    const footer = generateTableFooter(optionList, options);

    return `
${createHeading(content, 2)}
| Argument |${anyAlias ? ' Alias |' : ''} Type |${anyDescription ? ' Description |' : ''}
|-|${anyAlias ? '-|' : ''}-|${anyDescription ? '-|' : ''}
${optionList
    .map((option) => mapDefinitionDetails(option, options))
    .map((option) => createOptionRow(option, anyAlias, anyDescription))
    .join('\n')}
${footer != null ? footer + '\n' : ''}`;
}

function filterOptions(option: CommandLineOption, groups?: string | string[]): boolean {
    return (
        groups == null ||
        (typeof groups === 'string' && (groups === option.group || (groups === '_none' && option.group == null))) ||
        (Array.isArray(groups) &&
            (groups.some((group) => group === option.group) ||
                (groups.some((group) => group === '_none') && option.group == null)))
    );
}

export function createHeading(section: SectionHeader, defaultLevel: HeaderLevel): string {
    if (section.header == null) {
        return '';
    }

    const headingLevel = Array.from({ length: section.headerLevel || defaultLevel })
        .map(() => `#`)
        .join('');

    return `${headingLevel} ${section.header}
`;
}

export function createOptionRow(option: CommandLineOption, includeAlias = true, includeDescription = true): string {
    const alias = includeAlias ? ` ${option.alias == null ? '' : '**' + option.alias + '** '}|` : ``;
    const description = includeDescription
        ? ` ${option.description == null ? '' : convertChalkStringToMarkdown(option.description) + ' '}|`
        : ``;
    return `| **${option.name}** |${alias} ${getType(option)}|${description}`;
}

export function getType(option: CommandLineOption): string {
    if (option.typeLabel) {
        return `${convertChalkStringToMarkdown(option.typeLabel)} `;
    }

    //TODO: add modifiers

    const type = option.type ? option.type.name.toLowerCase() : 'string';
    const multiple = option.multiple || option.lazyMultiple ? '[]' : '';

    return `${type}${multiple} `;
}

export function generateUsageGuides(args: IWriteMarkDown): string[] | undefined {
    if (args.jsFile == null) {
        console.log(
            `No jsFile defined for usage guide generation. See 'write-markdown -h' for details on generating usage guides.`,
        );
        return undefined;
    }

    function mapJsImports(imports: JsImport[], jsFile: string) {
        return [...imports, ...args.configImportName.map((importName) => ({ jsFile, importName }))];
    }

    return args.jsFile
        .reduce(mapJsImports, new Array<JsImport>())
        .map(({ jsFile, importName }) => loadArgConfig(jsFile, importName))
        .filter(isDefined)
        .map(createUsageGuide);
}

export function loadArgConfig(jsFile: string, importName: string): UsageGuideConfig | undefined {
    const jsPath = join(process.cwd(), jsFile);
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const jsExports = require(jsPath);

    const argConfig: UsageGuideConfig = jsExports[importName];

    if (argConfig == null) {
        console.warn(`Could not import ArgumentConfig named '${importName}' from jsFile '${jsFile}'`);
        return undefined;
    }

    return argConfig;
}

function isDefined<T>(value: T | undefined | null): value is T {
    return value != null;
}
