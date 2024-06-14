import { ParseOptions, OptionContent, CommandLineOption, PropertyOptions, Content, OptionList } from '../contracts';
export declare function getOptionSections(options: ParseOptions<any>): OptionContent[];
export declare function getOptionFooterSection<T>(optionList: CommandLineOption<T>[], options: ParseOptions<any>): Content[];
export declare function generateTableFooter<T>(optionList: CommandLineOption<T>[], options: ParseOptions<any>): string | undefined;
export declare function addOptions<T>(content: OptionContent, optionList: CommandLineOption<T>[], options: ParseOptions<T>): OptionList;
/**
 * adds default or optional modifiers to type label or description
 * @param option
 */
export declare function mapDefinitionDetails<T>(definition: CommandLineOption<T>, options: ParseOptions<T>): CommandLineOption<T>;
export declare function isBoolean<T>(option: PropertyOptions<T>): boolean;
