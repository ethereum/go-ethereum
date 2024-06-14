import { IReplaceOptions } from '../contracts';
/**
 * Adds or replaces content between 2 markers within a text string
 * @param inputString
 * @param content
 * @param options
 * @returns
 */
export declare function addContent(inputString: string, content: string | string[], options: IReplaceOptions): string;
export declare function addCommandLineArgsFooter(fileContent: string): string;
