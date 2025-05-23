/* eslint-disable no-useless-escape */
import { convertChalkStringToMarkdown, removeAdditionalFormatting } from './string.helper';

describe('string.helper', () => {
    describe('convertChalkStringToMarkdown', () => {
        it('should remove unsupported chalk formatting', () => {
            expect(convertChalkStringToMarkdown(`some {underline modified underlined} text`)).toEqual(
                `some modified underlined text`,
            );
        });

        it('should replace bold formatting', () => {
            expect(convertChalkStringToMarkdown(`some {bold modified bold} text`)).toEqual(
                `some **modified bold** text`,
            );
        });

        it('should replace italic formatting', () => {
            expect(convertChalkStringToMarkdown(`some {italic modified italic} text`)).toEqual(
                `some *modified italic* text`,
            );
        });

        it('should replace bold italic formatting', () => {
            expect(convertChalkStringToMarkdown(`some {bold.italic modified bold italic} text`)).toEqual(
                `some ***modified bold italic*** text`,
            );
        });

        it('should replace italic bold formatting', () => {
            expect(convertChalkStringToMarkdown(`some {italic.bold modified italic bold} text`)).toEqual(
                `some ***modified italic bold*** text`,
            );
        });

        it('should replace highlight formatting', () => {
            expect(convertChalkStringToMarkdown(`some {highlight modified highlighted} text`)).toEqual(
                `some \`modified highlighted\` text`,
            );
        });

        it('should replace code formatting', () => {
            expect(convertChalkStringToMarkdown(`some {code modified code} text`)).toEqual(`some   
\`\`\`  
modified code  
\`\`\`  
 text`);
        });

        it('should replace code formatting with language', () => {
            expect(convertChalkStringToMarkdown(`some {code.typescript modified code} text`)).toEqual(`some   
\`\`\`typescript  
modified code  
\`\`\`  
 text`);
        });

        it('should add 2 blank spaces to new lines', () => {
            expect(
                convertChalkStringToMarkdown(`some text
over 2 lines`),
            ).toEqual(
                `some text  
over 2 lines`,
            );
        });
    });

    describe('removeAdditionalFormatting', () => {
        it('should leave existing chalk formatting', () => {
            expect(removeAdditionalFormatting(`some {underline modified underlined} text`)).toEqual(
                `some {underline modified underlined} text`,
            );
        });

        it('should replace highlight modifier', () => {
            expect(removeAdditionalFormatting(`some {highlight modified highlighted} and {bold bold} text`)).toEqual(
                `some modified highlighted and {bold bold} text`,
            );
        });

        it('should replace code modifier with curly braces', () => {
            expect(removeAdditionalFormatting(`some {code function()\{doSomething();\}} text`)).toEqual(
                `some function()\{doSomething();\} text`,
            );
        });

        it('should replace code modifier with curly braces and new lines', () => {
            expect(
                removeAdditionalFormatting(
                    `some {code function logMessage(message: string) \\{console.log(message);\\}} text`,
                ),
            ).toEqual(`some function logMessage(message: string) \\{console.log(message);\\} text`);
        });

        it('should replace code modifier with language with curly braces and new lines', () => {
            expect(
                removeAdditionalFormatting(
                    `some {code.typescript function logMessage(message: string) \\{console.log(message);\\}} text`,
                ),
            ).toEqual(`some function logMessage(message: string) \\{console.log(message);\\} text`);
        });
    });
});
