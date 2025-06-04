#!/usr/bin/env node

import { parse } from './parse';
import { IWriteMarkDown } from './contracts';
import { resolve, relative } from 'path';
import { readFileSync, writeFileSync } from 'fs';
import { addCommandLineArgsFooter, addContent, generateUsageGuides, insertCode } from './helpers';
import { argumentConfig, parseOptions } from './write-markdown.constants';
import format from 'string-format';
import chalk from 'chalk';

async function writeMarkdown() {
    const args = parse<IWriteMarkDown>(argumentConfig, parseOptions);

    const markdownPath = resolve(args.markdownPath);

    console.log(`Loading existing file from '${chalk.blue(markdownPath)}'`);
    const markdownFileContent = readFileSync(markdownPath).toString();

    const usageGuides = generateUsageGuides(args);

    let modifiedFileContent = markdownFileContent;

    if (usageGuides != null) {
        modifiedFileContent = addContent(markdownFileContent, usageGuides, args);
        if (!args.skipFooter) {
            modifiedFileContent = addCommandLineArgsFooter(modifiedFileContent);
        }
    }

    modifiedFileContent = await insertCode({ fileContent: modifiedFileContent, filePath: markdownPath }, args);

    const action = args.verify === true ? `verify` : `write`;
    const contentMatch = markdownFileContent === modifiedFileContent ? `match` : `nonMatch`;

    const relativePath = relative(process.cwd(), markdownPath);

    switch (`${action}_${contentMatch}`) {
        case 'verify_match':
            console.log(chalk.green(`'${relativePath}' content as expected. No update required.`));
            break;
        case 'verify_nonMatch':
            console.warn(
                chalk.yellow(
                    format(
                        args.verifyMessage || `'${relativePath}' file out of date. Rerun write-markdown to update.`,
                        {
                            fileName: relativePath,
                        },
                    ),
                ),
            );
            return process.exit(1);
        case 'write_match':
            console.log(chalk.blue(`'${relativePath}' content not modified, not writing to file.`));
            break;
        case 'write_nonMatch':
            console.log(`Writing modified file to '${chalk.blue(relativePath)}'`);
            writeFileSync(relativePath, modifiedFileContent);
            break;
    }
}

writeMarkdown();
