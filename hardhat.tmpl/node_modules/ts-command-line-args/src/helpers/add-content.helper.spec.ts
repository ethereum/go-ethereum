import { addContent } from './add-content.helper';
import { IReplaceOptions } from '../contracts';

describe('content.helper', () => {
    describe('addContent', () => {
        const newContent = `new content line one
new content line two`;

        let config: IReplaceOptions;

        beforeEach(() => {
            config = {
                replaceAbove: '##replaceAbove',
                replaceBelow: '##replaceBelow',
                removeDoubleBlankLines: false,
            };
        });

        it('should replace whole content when no markers found', () => {
            const initial = `content line 1
content line 2`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(newContent);
        });

        it('should add content at the end when replaceBelow found at end of content', () => {
            const initial = `content line 1
content line 2
##replaceBelow`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`content line 1
content line 2
##replaceBelow
new content line one
new content line two`);
        });

        it('should replace content at the end when replaceBelow found mid-content', () => {
            const initial = `content line 1
##replaceBelow
content line 2`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`content line 1
##replaceBelow
new content line one
new content line two`);
        });

        it('should add content at the top when replace above found at top of content', () => {
            const initial = `##replaceAbove
content line 1
content line 2`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`new content line one
new content line two
##replaceAbove
content line 1
content line 2`);
        });

        it('should replace content at the top when replace above found mid-document', () => {
            const initial = `content line 1
##replaceAbove
content line 2`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`new content line one
new content line two
##replaceAbove
content line 2`);
        });

        it('should add content between markers when no content exists already', () => {
            const initial = `content line 1
##replaceBelow
##replaceAbove
content line 2`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`content line 1
##replaceBelow
new content line one
new content line two
##replaceAbove
content line 2`);
        });

        it('should replace content between markers when content already exists', () => {
            const initial = `content line 1
##replaceBelow
content line 2
##replaceAbove
content line 3`;
            const result = addContent(initial, newContent, config);

            expect(result).toBe(`content line 1
##replaceBelow
new content line one
new content line two
##replaceAbove
content line 3`);
        });

        it('should replace content between markers when content already exists and an array of content passed in', () => {
            const initial = `content line 1
##replaceBelow
content line 2
##replaceAbove
content line 3`;
            const result = addContent(
                initial,
                [
                    newContent,
                    `other new content line one
other new content line two`,
                ],
                config,
            );

            expect(result).toBe(`content line 1
##replaceBelow
new content line one
new content line two
other new content line one
other new content line two
##replaceAbove
content line 3`);
        });

        it('should throw an error if add below appears above add above', () => {
            const initial = `content line 1
##replaceAbove
##replaceBelow
content line 3`;
            expect(() => addContent(initial, newContent, config)).toThrowError(
                `The replaceAbove marker '##replaceAbove' was found before the replaceBelow marker '##replaceBelow'. The replaceBelow marked must be before the replaceAbove.`,
            );
        });

        it('should not remove empty lines', () => {
            const initial = `content line 1


content line 2
##replaceBelow
##replaceAbove
content line 3`;
            const result = addContent(
                initial,
                `new content line one



new content line two`,
                config,
            );

            expect(result).toBe(`content line 1


content line 2
##replaceBelow
new content line one



new content line two
##replaceAbove
content line 3`);
        });

        it('should remove empty lines when passed in config', () => {
            const initial = `content line 1


content line 2
##replaceBelow
##replaceAbove
content line 3`;
            const result = addContent(
                initial,
                `new content line one



new content line two`,
                { ...config, removeDoubleBlankLines: true },
            );

            expect(result).toBe(`content line 1

content line 2
##replaceBelow
new content line one

new content line two
##replaceAbove
content line 3`);
        });
    });
});
