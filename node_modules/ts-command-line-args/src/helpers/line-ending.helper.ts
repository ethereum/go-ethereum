import { EOL } from 'os';

export type LineEnding = 'LF' | 'CRLF' | 'CR' | 'LFCR';
export const lineEndings: LineEnding[] = ['LF', 'CRLF', 'CR', 'LFCR'];

const multiCharRegExp = /(\r\n)|(\n\r)/g;
const singleCharRegExp = /(\r)|(\n)/g;

// https://en.wikipedia.org/wiki/Newline
/**
 * returns escape sequence for given line ending
 * @param lineEnding
 */
export function getEscapeSequence(lineEnding: LineEnding): string {
    switch (lineEnding) {
        case 'CR':
            return '\r';
        case 'CRLF':
            return '\r\n';
        case 'LF':
            return '\n';
        case 'LFCR':
            return '\n\r';
        default:
            return handleNever(lineEnding);
    }
}

function handleNever(lineEnding: never): string {
    throw new Error(`Unknown line ending: '${lineEnding}'. Line Ending must be one of ${lineEndings.join(', ')}`);
}

export function findEscapeSequence(content: string): string {
    const multiCharMatch = multiCharRegExp.exec(content);
    if (multiCharMatch != null) {
        return multiCharMatch[0];
    }
    const singleCharMatch = singleCharRegExp.exec(content);
    if (singleCharMatch != null) {
        return singleCharMatch[0];
    }
    return EOL;
}

/**
 * Splits a string into an array of lines.
 * Handles any line endings including a file with a mixture of lineEndings
 *
 * @param content multi line string to be split
 */
export function splitContent(content: string): string[] {
    content = content.replace(multiCharRegExp, '\n');
    content = content.replace(singleCharRegExp, '\n');
    return content.split('\n');
}

const nonWhitespaceRegExp = /[^ \t]/;

export function filterDoubleBlankLines(line: string, index: number, lines: string[]): boolean {
    const previousLine = index > 0 ? lines[index - 1] : undefined;

    return nonWhitespaceRegExp.test(line) || previousLine == null || nonWhitespaceRegExp.test(previousLine);
}
