import { FileDescription } from '../typechain/types';
/**
 * returns barrel files with reexports for all given paths
 *
 * @see https://github.com/basarat/typescript-book/blob/master/docs/tips/barrel.md
 */
export declare function createBarrelFiles(paths: string[], { typeOnly, postfix, moduleSuffix }: {
    typeOnly: boolean;
    postfix?: string;
    moduleSuffix?: string;
}): FileDescription[];
