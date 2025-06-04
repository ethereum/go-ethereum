import { LineEnding, getEscapeSequence, splitContent, findEscapeSequence } from './line-ending.helper';
import { EOL } from 'os';

describe('line-ending.helper', () => {
    const tests: { ending: LineEnding; escape: string }[] = [
        { ending: 'CR', escape: '\r' },
        { ending: 'CRLF', escape: '\r\n' },
        { ending: 'LF', escape: '\n' },
        { ending: 'LFCR', escape: '\n\r' },
    ];

    let lines: string[];

    beforeEach(() => {
        lines = ['line one', 'line two', 'line three', 'line four', 'line five'];
    });

    describe('getEscapeSequence', () => {
        tests.forEach((test) => {
            it(`should return correct escape sequence for ${test.ending}`, () => {
                expect(getEscapeSequence(test.ending)).toEqual(test.escape);
            });
        });

        it('should throw error for unknown line ending', () => {
            expect(() => getEscapeSequence('notEnding' as any)).toThrowError(
                `Unknown line ending: 'notEnding'. Line Ending must be one of LF, CRLF, CR, LFCR`,
            );
        });
    });

    describe('findEscapeSequence', () => {
        tests.forEach((test) => {
            it(`should return correct escape sequence for content using ${test.ending} endings`, () => {
                const content = lines.join(test.escape);
                expect(findEscapeSequence(content)).toEqual(test.escape);
            });
        });

        it('should return os default sequence when no line endings found', () => {
            expect(findEscapeSequence('')).toEqual(EOL);
        });
    });

    describe('splitContent', () => {
        tests.forEach((test) => {
            it(`should split file with ${test.ending} line endings correctly`, () => {
                const content = lines.join(test.escape);
                const result = splitContent(content);

                expect(result).toEqual(lines);
            });
        });

        it(`should split file with mixed line endings correctly`, () => {
            const content = `line one\nline two\n\rline three\rline four\r\nline five`;
            const result = splitContent(content);

            expect(result).toEqual(lines);
        });
    });
});
