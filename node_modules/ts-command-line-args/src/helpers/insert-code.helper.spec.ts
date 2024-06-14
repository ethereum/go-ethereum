import { IInsertCodeOptions } from '../contracts';
import {
    insertCodeBelowDefault,
    insertCodeAboveDefault,
    copyCodeBelowDefault,
    copyCodeAboveDefault,
} from '../write-markdown.constants';
import { insertCode } from './insert-code.helper';
import * as originalFs from 'fs';
import { any, IMocked, Mock, registerMock, reset, setupFunction } from '@morgan-stanley/ts-mocking-bird';
import { EOL } from 'os';
import { resolve, join } from 'path';

const beforeInsertionLine = `beforeInsertion`;
const afterInsertionLine = `afterInsertion`;

const insertLineOne = `insertLineOne`;
const insertLineTwo = `insertLineTwo`;

let insertBelowToken = `${insertCodeBelowDefault} file="someFile.ts" )`;

const sampleDirName = `sample/dirname`;

// eslint-disable-next-line @typescript-eslint/no-var-requires
jest.mock('fs', () => require('@morgan-stanley/ts-mocking-bird').proxyJestModule(require.resolve('fs')));

describe(`(${insertCode.name}) insert-code.helper`, () => {
    let mockedFs: IMocked<typeof originalFs>;
    let insertCodeFromContent: string;

    beforeEach(() => {
        insertBelowToken = `${insertCodeBelowDefault} file="someFile.ts" )`;
        insertCodeFromContent = `${insertLineOne}${EOL}${insertLineTwo}`;

        mockedFs = Mock.create<typeof originalFs>().setup(
            setupFunction('readFile', ((_path: string, callback: (err: Error | null, data: Buffer) => void) => {
                callback(null, Buffer.from(insertCodeFromContent));
            }) as any),
            setupFunction('writeFile', ((_path: string, _data: any, callback: () => void) => {
                callback();
            }) as any),
        );

        registerMock(originalFs, mockedFs.mock);
    });

    afterEach(() => {
        reset(originalFs);
    });

    function createOptions(partialOptions?: Partial<IInsertCodeOptions>): IInsertCodeOptions {
        return {
            insertCodeBelow: insertCodeBelowDefault,
            insertCodeAbove: insertCodeAboveDefault,
            copyCodeBelow: copyCodeBelowDefault,
            copyCodeAbove: copyCodeAboveDefault,
            removeDoubleBlankLines: false,
            ...partialOptions,
        };
    }

    it(`should return original string when no insertBelow token provided`, async () => {
        const fileContent = [beforeInsertionLine, afterInsertionLine].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions({ insertCodeAbove: undefined, insertCodeBelow: undefined }),
        );

        expect(result).toEqual(fileContent);
    });

    it(`should return original string when no insertBelow token found`, async () => {
        const fileContent = [beforeInsertionLine, afterInsertionLine].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(result).toEqual(fileContent);
    });

    it(`should insert all file content with default tokens`, async () => {
        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertLineTwo,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert all file content when passed a file path`, async () => {
        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        mockedFs.setupFunction('readFile', ((path: string, callback: (err: Error | null, data: Buffer) => void) => {
            if (path.indexOf(`originalFilePath.ts`) > 0) {
                callback(null, Buffer.from(fileContent));
            } else {
                callback(null, Buffer.from(`${insertLineOne}${EOL}${insertLineTwo}`));
            }
        }) as any);

        const result = await insertCode(`${sampleDirName}/originalFilePath.ts`, createOptions());

        expect(
            mockedFs.withFunction('readFile').withParameters(resolve(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();
        expect(
            mockedFs.withFunction('readFile').withParameters(resolve(`${sampleDirName}/originalFilePath.ts`), any()),
        ).wasCalledOnce();
        expect(
            mockedFs
                .withFunction('writeFile')
                .withParameters(resolve(`${sampleDirName}/originalFilePath.ts`), any(), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertLineTwo,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should remove double blank lines if set to true`, async () => {
        const fileContent = [
            beforeInsertionLine,
            insertBelowToken,
            '',
            '',
            '',
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions({ removeDoubleBlankLines: true }),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertLineTwo,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert all file content with custom tokens`, async () => {
        const fileContent = [
            beforeInsertionLine,
            `customInsertAfterToken file="somePath"`,
            `customInsertBeforeToken`,
            afterInsertionLine,
        ].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions({ insertCodeBelow: `customInsertAfterToken`, insertCodeAbove: `customInsertBeforeToken` }),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'somePath'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            `customInsertAfterToken file="somePath"`,
            insertLineOne,
            insertLineTwo,
            `customInsertBeforeToken`,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should remove end of file if no insertAbove token`, async () => {
        const fileContent = [beforeInsertionLine, insertBelowToken, afterInsertionLine].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [beforeInsertionLine, insertBelowToken, insertLineOne, insertLineTwo].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should throw error if insertBelow token provided with no file`, async () => {
        const fileContent = [
            beforeInsertionLine,
            insertCodeBelowDefault,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        let error: Error | undefined;

        try {
            await insertCode({ fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` }, createOptions());
        } catch (e: any) {
            error = e;
        }

        expect(error?.message).toEqual(
            `insert code token ([//]: # (ts-command-line-args_write-markdown_insertCodeBelow) found in file but file path not specified (file="relativePath/from/markdown/toFile.whatever")`,
        );
    });

    it(`should should only insert file content between copyAbove and copyBelow tokens`, async () => {
        insertCodeFromContent = [
            'randomFirstLine',
            copyCodeBelowDefault,
            insertLineOne,
            copyCodeAboveDefault,
            insertLineTwo,
        ].join('\n');
        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert selected snippet when snippet defined`, async () => {
        insertCodeFromContent = [
            'randomFirstLine',
            `// ts-command-line-args_write-markdown_copyCodeBelow expectedSnippet`,
            insertLineOne,
            copyCodeAboveDefault,
            copyCodeBelowDefault,
            insertLineTwo,
            copyCodeAboveDefault,
        ].join('\n');
        insertBelowToken = `${insertCodeBelowDefault} file="someFile.ts" snippetName="expectedSnippet" )`;

        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should should only insert file content after copyBelow token`, async () => {
        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        const fileLines = [insertLineOne, copyCodeBelowDefault, insertLineTwo];

        mockedFs.setupFunction('readFile', ((_path: string, callback: (err: Error | null, data: Buffer) => void) => {
            callback(null, Buffer.from(fileLines.join(EOL)));
        }) as any);

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineTwo,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should should only insert file content above copyAbove token`, async () => {
        const fileContent = [beforeInsertionLine, insertBelowToken, insertCodeAboveDefault, afterInsertionLine].join(
            '\n',
        );

        const fileLines = [insertLineOne, copyCodeAboveDefault, insertLineTwo];

        mockedFs.setupFunction('readFile', ((_path: string, callback: (err: Error | null, data: Buffer) => void) => {
            callback(null, Buffer.from(fileLines.join(EOL)));
        }) as any);

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            insertBelowToken,
            insertLineOne,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert a code comment`, async () => {
        const fileContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="someFile.ts" codeComment )`,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="someFile.ts" codeComment )`,
            '```',
            insertLineOne,
            insertLineTwo,
            '```',
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert a name code comment`, async () => {
        const fileContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="someFile.ts" codeComment="ts" )`,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        const result = await insertCode(
            { fileContent, filePath: `${sampleDirName}/'originalFilePath.ts` },
            createOptions(),
        );

        expect(
            mockedFs.withFunction('readFile').withParameters(join(sampleDirName, 'someFile.ts'), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="someFile.ts" codeComment="ts" )`,
            '```ts',
            insertLineOne,
            insertLineTwo,
            '```',
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });

    it(`should insert content from 2 different files in 2 different locations`, async () => {
        const inBetweenFilesLine = 'in  between files';
        const fileContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="insertFileOne.ts" )`,
            insertCodeAboveDefault,
            inBetweenFilesLine,
            `${insertCodeBelowDefault} file="insertFileTwo.ts" )`,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        mockedFs.setupFunction('readFile', ((path: string, callback: (err: Error | null, data: Buffer) => void) => {
            if (path.indexOf(`originalFilePath.ts`) > 0) {
                callback(null, Buffer.from(fileContent));
            } else if (path.indexOf(`insertFileOne.ts`) > 0) {
                callback(null, Buffer.from(`fileOneLineOne${EOL}fileOneLineTwo`));
            } else if (path.indexOf(`insertFileTwo.ts`) > 0) {
                callback(null, Buffer.from(`fileTwoLineOne${EOL}fileTwoLineTwo`));
            } else {
                throw new Error(`unknown file path: ${path}`);
            }
        }) as any);

        const result = await insertCode(`${sampleDirName}/'originalFilePath.ts`, createOptions());

        expect(
            mockedFs.withFunction('readFile').withParameters(resolve(sampleDirName, 'insertFileOne.ts'), any()),
        ).wasCalledOnce();
        expect(
            mockedFs.withFunction('readFile').withParameters(resolve(sampleDirName, 'insertFileTwo.ts'), any()),
        ).wasCalledOnce();
        expect(
            mockedFs.withFunction('readFile').withParameters(resolve(`${sampleDirName}/'originalFilePath.ts`), any()),
        ).wasCalledOnce();

        const expectedContent = [
            beforeInsertionLine,
            `${insertCodeBelowDefault} file="insertFileOne.ts" )`,
            `fileOneLineOne`,
            `fileOneLineTwo`,
            insertCodeAboveDefault,
            inBetweenFilesLine,
            `${insertCodeBelowDefault} file="insertFileTwo.ts" )`,
            `fileTwoLineOne`,
            `fileTwoLineTwo`,
            insertCodeAboveDefault,
            afterInsertionLine,
        ].join('\n');

        expect(result).toEqual(expectedContent);
    });
});
