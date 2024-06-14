import {
    ParseOptions,
    OptionContent,
    CommandLineOption,
    OptionalProperty,
    PropertyOptions,
    Content,
    OptionList,
} from '../contracts';

export function getOptionSections(options: ParseOptions<any>): OptionContent[] {
    return (
        options.optionSections || [
            { header: options.optionsHeaderText || 'Options', headerLevel: options.optionsHeaderLevel || 2 },
        ]
    );
}

export function getOptionFooterSection<T>(optionList: CommandLineOption<T>[], options: ParseOptions<any>): Content[] {
    const optionsFooter = generateTableFooter(optionList, options);

    if (optionsFooter != null) {
        console.log(`Adding footer: ${optionsFooter}`);
        return [{ content: optionsFooter }];
    }

    return [];
}

export function generateTableFooter<T>(
    optionList: CommandLineOption<T>[],
    options: ParseOptions<any>,
): string | undefined {
    if (options.addOptionalDefaultExplanatoryFooter != true || options.displayOptionalAndDefault != true) {
        return undefined;
    }

    const optionalProps = optionList.some((option) => (option as unknown as OptionalProperty).optional === true);
    const defaultProps = optionList.some((option) => option.defaultOption === true);

    if (optionalProps || defaultProps) {
        const footerValues = [
            optionalProps != null ? '(O) = optional' : undefined,
            defaultProps != null ? '(D) = default option' : null,
        ];
        return footerValues.filter((v) => v != null).join(', ');
    }

    return undefined;
}

export function addOptions<T>(
    content: OptionContent,
    optionList: CommandLineOption<T>[],
    options: ParseOptions<T>,
): OptionList {
    optionList = optionList.map((option) => mapDefinitionDetails(option, options));

    return { ...content, optionList };
}

/**
 * adds default or optional modifiers to type label or description
 * @param option
 */
export function mapDefinitionDetails<T>(
    definition: CommandLineOption<T>,
    options: ParseOptions<T>,
): CommandLineOption<T> {
    definition = mapOptionTypeLabel(definition, options);
    definition = mapOptionDescription(definition, options);

    return definition;
}

function mapOptionDescription<T>(definition: CommandLineOption<T>, options: ParseOptions<T>): CommandLineOption<T> {
    if (options.prependParamOptionsToDescription !== true || isBoolean(definition)) {
        return definition;
    }

    definition.description = definition.description || '';

    if (definition.defaultOption) {
        definition.description = `Default Option. ${definition.description}`;
    }

    if ((definition as unknown as OptionalProperty).optional === true) {
        definition.description = `Optional. ${definition.description}`;
    }

    if (definition.defaultValue != null) {
        definition.description = `Defaults to ${JSON.stringify(definition.defaultValue)}. ${definition.description}`;
    }

    return definition;
}

function mapOptionTypeLabel<T>(definition: CommandLineOption<T>, options: ParseOptions<T>): CommandLineOption<T> {
    if (options.displayOptionalAndDefault !== true || isBoolean(definition)) {
        return definition;
    }

    definition.typeLabel = definition.typeLabel || getTypeLabel(definition);

    if (definition.defaultOption) {
        definition.typeLabel = `${definition.typeLabel} (D)`;
    }

    if ((definition as unknown as OptionalProperty).optional === true) {
        definition.typeLabel = `${definition.typeLabel} (O)`;
    }

    return definition;
}

function getTypeLabel<T>(definition: CommandLineOption<T>) {
    let typeLabel = definition.type ? definition.type.name.toLowerCase() : 'string';
    const multiple = definition.multiple || definition.lazyMultiple ? '[]' : '';
    if (typeLabel) {
        typeLabel = typeLabel === 'boolean' ? '' : `{underline ${typeLabel}${multiple}}`;
    }

    return typeLabel;
}

export function isBoolean<T>(option: PropertyOptions<T>): boolean {
    return option.type.name === 'Boolean';
}
