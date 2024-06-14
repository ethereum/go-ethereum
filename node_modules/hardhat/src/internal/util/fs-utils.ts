import fsPromises from "fs/promises";
import fs from "fs";
import path from "path";
import { CustomError } from "../core/errors";

// We use this error to encapsulate any other error possibly thrown by node's
// fs apis, as sometimes their errors don't have stack traces.
export class FileSystemAccessError extends CustomError {}

export class FileNotFoundError extends CustomError {
  constructor(filePath: string, parent?: Error) {
    super(`File ${filePath} not found`, parent);
  }
}
export class InvalidDirectoryError extends CustomError {
  constructor(filePath: string, parent: Error) {
    super(`Invalid directory ${filePath}`, parent);
  }
}

/**
 * Returns the real path of absolutePath, resolving symlinks.
 *
 * @throws FileNotFoundError if absolutePath doesn't exist.
 */
export async function getRealPath(absolutePath: string): Promise<string> {
  try {
    // This method returns the actual casing.
    // Please read Node.js' docs to learn more.
    return await fsPromises.realpath(path.normalize(absolutePath));
  } catch (e: any) {
    if (e.code === "ENOENT") {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw new FileNotFoundError(absolutePath, e);
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileSystemAccessError(e.message, e);
  }
}

/**
 * Sync version of getRealPath
 *
 * @see getRealCase
 */
export function getRealPathSync(absolutePath: string): string {
  try {
    // This method returns the actual casing.
    // Please read Node.js' docs to learn more.
    return fs.realpathSync.native(path.normalize(absolutePath));
  } catch (e: any) {
    if (e.code === "ENOENT") {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw new FileNotFoundError(absolutePath, e);
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileSystemAccessError(e.message, e);
  }
}

/**
 * Returns an array of files (not dirs) that match a condition.
 *
 * @param absolutePathToDir A directory. If it doesn't exist `[]` is returned.
 * @param matches A function to filter files (not directories)
 * @returns An array of absolute paths. Each file has its true case, except
 *  for the initial absolutePathToDir part, which preserves the given casing.
 *  No order is guaranteed.
 */
export async function getAllFilesMatching(
  absolutePathToDir: string,
  matches?: (absolutePathToFile: string) => boolean
): Promise<string[]> {
  const dir = await readdir(absolutePathToDir);

  const results = await Promise.all(
    dir.map(async (file) => {
      const absolutePathToFile = path.join(absolutePathToDir, file);
      const stats = await fsPromises.stat(absolutePathToFile);
      if (stats.isDirectory()) {
        const files = await getAllFilesMatching(absolutePathToFile, matches);
        return files.flat();
      } else if (matches === undefined || matches(absolutePathToFile)) {
        return absolutePathToFile;
      } else {
        return [];
      }
    })
  );

  return results.flat();
}

/**
 * Sync version of getAllFilesMatching
 *
 * @see getAllFilesMatching
 */
export function getAllFilesMatchingSync(
  absolutePathToDir: string,
  matches?: (absolutePathToFile: string) => boolean
): string[] {
  const dir = readdirSync(absolutePathToDir);

  const results = dir.map((file) => {
    const absolutePathToFile = path.join(absolutePathToDir, file);
    const stats = fs.statSync(absolutePathToFile);
    if (stats.isDirectory()) {
      return getAllFilesMatchingSync(absolutePathToFile, matches).flat();
    } else if (matches === undefined || matches(absolutePathToFile)) {
      return absolutePathToFile;
    } else {
      return [];
    }
  });

  return results.flat();
}

/**
 * Returns the true case relative path of `relativePath` from `from`, without
 * resolving symlinks.
 */
export async function getFileTrueCase(
  from: string,
  relativePath: string
): Promise<string> {
  const dirEntries = await readdir(from);

  const parts = relativePath.split(path.sep);
  const nextDirLowerCase = parts[0].toLowerCase();

  for (const dirEntry of dirEntries) {
    if (dirEntry.toLowerCase() === nextDirLowerCase) {
      if (parts.length === 1) {
        return dirEntry;
      }

      return path.join(
        dirEntry,
        await getFileTrueCase(
          path.join(from, dirEntry),
          path.relative(parts[0], relativePath)
        )
      );
    }
  }

  // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
  throw new FileNotFoundError(path.join(from, relativePath));
}

/**
 * Sync version of getFileTrueCase
 *
 * @see getFileTrueCase
 */
export function getFileTrueCaseSync(
  from: string,
  relativePath: string
): string {
  const dirEntries = readdirSync(from);

  const parts = relativePath.split(path.sep);
  const nextDirLowerCase = parts[0].toLowerCase();

  for (const dirEntry of dirEntries) {
    if (dirEntry.toLowerCase() === nextDirLowerCase) {
      if (parts.length === 1) {
        return dirEntry;
      }

      return path.join(
        dirEntry,
        getFileTrueCaseSync(
          path.join(from, dirEntry),
          path.relative(parts[0], relativePath)
        )
      );
    }
  }

  // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
  throw new FileNotFoundError(path.join(from, relativePath));
}

async function readdir(absolutePathToDir: string) {
  try {
    return await fsPromises.readdir(absolutePathToDir);
  } catch (e: any) {
    if (e.code === "ENOENT") {
      return [];
    }

    if (e.code === "ENOTDIR") {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw new InvalidDirectoryError(absolutePathToDir, e);
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileSystemAccessError(e.message, e);
  }
}

function readdirSync(absolutePathToDir: string) {
  try {
    return fs.readdirSync(absolutePathToDir);
  } catch (e: any) {
    if (e.code === "ENOENT") {
      return [];
    }

    if (e.code === "ENOTDIR") {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw new InvalidDirectoryError(absolutePathToDir, e);
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new FileSystemAccessError(e.message, e);
  }
}
