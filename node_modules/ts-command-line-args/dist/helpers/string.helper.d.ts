/**
 * Converts a string with chalk formatting into a string safe for markdown
 *
 * @param input
 */
export declare function convertChalkStringToMarkdown(input: string): string;
/**
 * Removes formatting not supported by chalk
 *
 * @param input
 */
export declare function removeAdditionalFormatting(input: string): string;
