import { IInsertCodeOptions } from '../contracts';
import { filterDoubleBlankLines, findEscapeSequence, splitContent } from './line-ending.helper';
import { isAbsolute, resolve, dirname, join } from 'path';
import { promisify } from 'util';
import { readFile, writeFile } from 'fs';
import chalk from 'chalk';

const asyncReadFile = promisify(readFile);
const asyncWriteFile = promisify(writeFile);

export type FileDetails = {
    filePath: string;
    fileContent: string;
};

/**
 * Loads content from other files and inserts it into the target file
 * @param input - if a string is provided the target file is loaded from that path AND saved to that path once content has been inserted. If a `FileDetails` object is provided the content is not saved when done.
 * @param partialOptions - optional. changes the default tokens
 */
export async function insertCode(
    input: FileDetails | string,
    partialOptions?: Partial<IInsertCodeOptions>,
): Promise<string> {
    const options: IInsertCodeOptions = { removeDoubleBlankLines: false, ...partialOptions };

    let fileDetails: FileDetails;

    if (typeof input === 'string') {
        const filePath = resolve(input);
        console.log(`Loading existing file from '${chalk.blue(filePath)}'`);
        fileDetails = { filePath, fileContent: (await asyncReadFile(filePath)).toString() };
    } else {
        fileDetails = input;
    }

    const content = fileDetails.fileContent;

    const lineBreak = findEscapeSequence(content);
    let lines = splitContent(content);

    lines = await insertCodeImpl(fileDetails.filePath, lines, options, 0);

    if (options.removeDoubleBlankLines) {
        lines = lines.filter((line, index, lines) => filterDoubleBlankLines(line, index, lines));
    }

    const modifiedContent = lines.join(lineBreak);

    if (typeof input === 'string') {
        console.log(`Saving modified content to '${chalk.blue(fileDetails.filePath)}'`);
        await asyncWriteFile(fileDetails.filePath, modifiedContent);
    }

    return modifiedContent;
}

async function insertCodeImpl(
    filePath: string,
    lines: string[],
    options: IInsertCodeOptions,
    startLine: number,
): Promise<string[]> {
    const insertCodeBelow = options?.insertCodeBelow;
    const insertCodeAbove = options?.insertCodeAbove;

    if (insertCodeBelow == null) {
        return Promise.resolve(lines);
    }

    const insertCodeBelowResult =
        insertCodeBelow != null
            ? findIndex(lines, (line) => line.indexOf(insertCodeBelow) === 0, startLine)
            : undefined;

    if (insertCodeBelowResult == null) {
        return Promise.resolve(lines);
    }

    const insertCodeAboveResult =
        insertCodeAbove != null
            ? findIndex(lines, (line) => line.indexOf(insertCodeAbove) === 0, insertCodeBelowResult.lineIndex)
            : undefined;

    const linesFromFile = await loadLines(filePath, options, insertCodeBelowResult);

    const linesBefore = lines.slice(0, insertCodeBelowResult.lineIndex + 1);
    const linesAfter = insertCodeAboveResult != null ? lines.slice(insertCodeAboveResult.lineIndex) : [];

    lines = [...linesBefore, ...linesFromFile, ...linesAfter];

    return insertCodeAboveResult == null
        ? lines
        : insertCodeImpl(filePath, lines, options, insertCodeAboveResult.lineIndex);
}

const fileRegExp = /file="([^"]+)"/;
const codeCommentRegExp = /codeComment(="([^"]+)")?/; //https://regex101.com/r/3MVdBO/1
const snippetRegExp = /snippetName="([^"]+)"/;

async function loadLines(
    targetFilePath: string,
    options: IInsertCodeOptions,
    result: FindLineResults,
): Promise<string[]> {
    const partialPathResult = fileRegExp.exec(result.line);

    if (partialPathResult == null) {
        throw new Error(
            `insert code token (${options.insertCodeBelow}) found in file but file path not specified (file="relativePath/from/markdown/toFile.whatever")`,
        );
    }
    const codeCommentResult = codeCommentRegExp.exec(result.line);
    const snippetResult = snippetRegExp.exec(result.line);
    const partialPath = partialPathResult[1];

    const filePath = isAbsolute(partialPath) ? partialPath : join(dirname(targetFilePath), partialPathResult[1]);
    console.log(`Inserting code from '${chalk.blue(filePath)}' into '${chalk.blue(targetFilePath)}'`);

    const fileBuffer = await asyncReadFile(filePath);

    let contentLines = splitContent(fileBuffer.toString());

    const copyBelowMarker = options.copyCodeBelow;
    const copyAboveMarker = options.copyCodeAbove;

    const copyBelowIndex =
        copyBelowMarker != null ? contentLines.findIndex(findLine(copyBelowMarker, snippetResult?.[1])) : -1;
    const copyAboveIndex =
        copyAboveMarker != null
            ? contentLines.findIndex((line, index) => line.indexOf(copyAboveMarker) === 0 && index > copyBelowIndex)
            : -1;

    if (snippetResult != null && copyBelowIndex < 0) {
        throw new Error(
            `The copyCodeBelow marker '${options.copyCodeBelow}' was not found with the requested snippet: '${snippetResult[1]}'`,
        );
    }

    contentLines = contentLines.slice(copyBelowIndex + 1, copyAboveIndex > 0 ? copyAboveIndex : undefined);

    if (codeCommentResult != null) {
        contentLines = ['```' + (codeCommentResult[2] ?? ''), ...contentLines, '```'];
    }

    return contentLines;
}

function findLine(copyBelowMarker: string, snippetName?: string): (line: string) => boolean {
    return (line: string): boolean => {
        return line.indexOf(copyBelowMarker) === 0 && (snippetName == null || line.indexOf(snippetName) > 0);
    };
}

type FindLineResults = { line: string; lineIndex: number };

function findIndex(
    lines: string[],
    predicate: (line: string) => boolean,
    startLine: number,
): FindLineResults | undefined {
    for (let lineIndex = startLine; lineIndex < lines.length; lineIndex++) {
        const line = lines[lineIndex];
        if (predicate(line)) {
            return { lineIndex, line };
        }
    }

    return undefined;
}
