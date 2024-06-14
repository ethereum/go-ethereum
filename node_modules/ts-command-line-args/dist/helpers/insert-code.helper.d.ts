import { IInsertCodeOptions } from '../contracts';
export declare type FileDetails = {
    filePath: string;
    fileContent: string;
};
/**
 * Loads content from other files and inserts it into the target file
 * @param input - if a string is provided the target file is loaded from that path AND saved to that path once content has been inserted. If a `FileDetails` object is provided the content is not saved when done.
 * @param partialOptions - optional. changes the default tokens
 */
export declare function insertCode(input: FileDetails | string, partialOptions?: Partial<IInsertCodeOptions>): Promise<string>;
