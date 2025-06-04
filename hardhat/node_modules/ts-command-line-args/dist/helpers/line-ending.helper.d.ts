export declare type LineEnding = 'LF' | 'CRLF' | 'CR' | 'LFCR';
export declare const lineEndings: LineEnding[];
/**
 * returns escape sequence for given line ending
 * @param lineEnding
 */
export declare function getEscapeSequence(lineEnding: LineEnding): string;
export declare function findEscapeSequence(content: string): string;
/**
 * Splits a string into an array of lines.
 * Handles any line endings including a file with a mixture of lineEndings
 *
 * @param content multi line string to be split
 */
export declare function splitContent(content: string): string[];
export declare function filterDoubleBlankLines(line: string, index: number, lines: string[]): boolean;
