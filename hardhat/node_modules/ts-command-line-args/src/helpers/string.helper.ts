const chalkStringStyleRegExp = /(?<!\\){([^}]+?)[ \n\r](.+?[^\\])}/gms;
const newLineRegExp = /\n/g;

const highlightModifier = 'highlight';
const codeModifier = 'code';

/**
 * Converts a string with chalk formatting into a string safe for markdown
 *
 * @param input
 */
export function convertChalkStringToMarkdown(input: string): string {
    return (
        input
            .replace(chalkStringStyleRegExp, replaceChalkFormatting)
            //replace new line with 2 spaces then new line
            .replace(newLineRegExp, '  \n')
            .replace(/\\{/g, '{')
            .replace(/\\}/g, '}')
    );
}

function replaceChalkFormatting(_substring: string, ...matches: string[]): string {
    let modifier = '';
    if (matches[0].indexOf(highlightModifier) >= 0) {
        modifier = '`';
    } else if (matches[0].indexOf(codeModifier) >= 0) {
        const codeOptions = matches[0].split('.');
        modifier = '\n```';
        if (codeOptions[1] != null) {
            return `${modifier}${codeOptions[1]}\n${matches[1]}${modifier}\n`;
        } else {
            return `${modifier}\n${matches[1]}${modifier}\n`;
        }
    } else {
        if (matches[0].indexOf('bold') >= 0) {
            modifier += '**';
        }
        if (matches[0].indexOf('italic') >= 0) {
            modifier += '*';
        }
    }
    return `${modifier}${matches[1]}${modifier}`;
}

/**
 * Removes formatting not supported by chalk
 *
 * @param input
 */
export function removeAdditionalFormatting(input: string): string {
    return input.replace(chalkStringStyleRegExp, removeNonChalkFormatting);
}

function removeNonChalkFormatting(substring: string, ...matches: string[]): string {
    const nonChalkFormats = [highlightModifier, codeModifier];

    if (nonChalkFormats.some((format) => matches[0].indexOf(format) >= 0)) {
        return matches[1];
    }

    return substring;
}
