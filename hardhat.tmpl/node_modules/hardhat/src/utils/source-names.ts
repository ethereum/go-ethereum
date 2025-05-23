import path from "path";

import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";
import { FileNotFoundError, getFileTrueCase } from "../internal/util/fs-utils";
import { getPackageName } from "../internal/util/packageInfo";

const NODE_MODULES = "node_modules";

/**
 * This function validates the source name's format.
 *
 * It throws if the format is invalid.
 * If it doesn't throw all you know is that the format is valid.
 */
export function validateSourceNameFormat(sourceName: string) {
  if (isAbsolutePathSourceName(sourceName)) {
    throw new HardhatError(
      ERRORS.SOURCE_NAMES.INVALID_SOURCE_NAME_ABSOLUTE_PATH,
      {
        name: sourceName,
      }
    );
  }

  if (isExplicitRelativePath(sourceName)) {
    throw new HardhatError(
      ERRORS.SOURCE_NAMES.INVALID_SOURCE_NAME_RELATIVE_PATH,
      {
        name: sourceName,
      }
    );
  }

  // We check this before normalizing so we are sure that the difference
  // comes from slash vs backslash
  if (replaceBackslashes(sourceName) !== sourceName) {
    throw new HardhatError(
      ERRORS.SOURCE_NAMES.INVALID_SOURCE_NAME_BACKSLASHES,
      {
        name: sourceName,
      }
    );
  }

  if (normalizeSourceName(sourceName) !== sourceName) {
    throw new HardhatError(ERRORS.SOURCE_NAMES.INVALID_SOURCE_NOT_NORMALIZED, {
      name: sourceName,
    });
  }
}

/**
 * This function returns true if the sourceName is, potentially, from a local
 * file. It doesn't validate that the file actually exists.
 *
 * The source name must be in a valid format.
 */
export async function isLocalSourceName(
  projectRoot: string,
  sourceName: string
): Promise<boolean> {
  // Note that we consider "hardhat/console.sol" as a special case here.
  // This lets someone have a "hardhat" directory within their project without
  // it impacting their use of `console.log`.
  // See issue https://github.com/nomiclabs/hardhat/issues/998
  if (
    sourceName.includes(NODE_MODULES) ||
    sourceName === "hardhat/console.sol"
  ) {
    return false;
  }

  const slashIndex = sourceName.indexOf("/");
  const firstDirOrFileName =
    slashIndex !== -1 ? sourceName.substring(0, slashIndex) : sourceName;

  try {
    await getFileTrueCase(projectRoot, firstDirOrFileName);
  } catch (error) {
    if (error instanceof FileNotFoundError) {
      return false;
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw error;
  }

  return true;
}

/**
 * Validates that a source name exists, starting from `fromDir`, and has the
 * right casing.
 *
 * The source name must be in a valid format.
 */
export async function validateSourceNameExistenceAndCasing(
  fromDir: string,
  sourceName: string
) {
  const trueCaseSourceName = await getSourceNameTrueCase(fromDir, sourceName);

  if (trueCaseSourceName !== sourceName) {
    throw new HardhatError(ERRORS.SOURCE_NAMES.WRONG_CASING, {
      incorrect: sourceName,
      correct: trueCaseSourceName,
    });
  }
}

/**
 * Returns the source name of an existing local file's absolute path.
 *
 * Throws is the file doesn't exist, it's not inside the project, or belongs
 * to a library.
 */
export async function localPathToSourceName(
  projectRoot: string,
  localFileAbsolutePath: string
): Promise<string> {
  const relativePath = path.relative(projectRoot, localFileAbsolutePath);
  const normalized = normalizeSourceName(relativePath);

  if (normalized.startsWith("..")) {
    throw new HardhatError(ERRORS.SOURCE_NAMES.EXTERNAL_AS_LOCAL, {
      path: localFileAbsolutePath,
    });
  }

  if (normalized.includes(NODE_MODULES)) {
    throw new HardhatError(ERRORS.SOURCE_NAMES.NODE_MODULES_AS_LOCAL, {
      path: localFileAbsolutePath,
    });
  }

  return getSourceNameTrueCase(projectRoot, relativePath);
}

/**
 * This function takes a valid local source name and returns its path. The
 * source name doesn't need to point to an existing file.
 */
export function localSourceNameToPath(
  projectRoot: string,
  sourceName: string
): string {
  return path.join(projectRoot, sourceName);
}

/**
 * Normalizes the source name, for example, by replacing `a/./b` with `a/b`.
 *
 * The sourceName param doesn't have to be a valid source name. It can,
 * for example, be denormalized.
 */
export function normalizeSourceName(sourceName: string): string {
  return replaceBackslashes(path.normalize(sourceName));
}

/**
 * This function returns true if the sourceName is a unix absolute path or a
 * platform-dependent one.
 *
 * This function is used instead of just `path.isAbsolute` to ensure that
 * source names never start with `/`, even on Windows.
 */
export function isAbsolutePathSourceName(sourceName: string): boolean {
  return path.isAbsolute(sourceName) || sourceName.startsWith("/");
}

/**
 * This function returns true if the sourceName is a unix path that is based on
 * the current directory `./`.
 */
function isExplicitRelativePath(sourceName: string): boolean {
  const [base] = sourceName.split("/", 1);
  return base === "." || base === "..";
}

/**
 * This function replaces backslashes (\\) with slashes (/).
 *
 * Note that a source name must not contain backslashes.
 */
export function replaceBackslashes(str: string): string {
  // Based in the npm module slash
  const isExtendedLengthPath = /^\\\\\?\\/.test(str);
  const hasNonAscii = /[^\u0000-\u0080]+/.test(str);

  if (isExtendedLengthPath || hasNonAscii) {
    return str;
  }

  return str.replace(/\\/g, "/");
}

function slashesToPathSeparator(str: string): string {
  if (path.sep === "/") {
    return str;
  }

  return str.replace(/\//g, path.sep);
}

/**
 * Returns the true casing of `p` as a relative path from `fromDir`. Throws if
 * `p` doesn't exist. `p` MUST be in source name format.
 */
async function getSourceNameTrueCase(
  fromDir: string,
  p: string
): Promise<string> {
  try {
    const realCase = await getFileTrueCase(fromDir, slashesToPathSeparator(p));
    return normalizeSourceName(realCase);
  } catch (error) {
    if (error instanceof FileNotFoundError) {
      throw new HardhatError(
        ERRORS.SOURCE_NAMES.FILE_NOT_FOUND,
        {
          name: p,
        },
        error
      );
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw error;
  }
}

/**
 * This function returns true if the sourceName contains the current package's name
 * as a substring
 */
export async function includesOwnPackageName(
  sourceName: string
): Promise<boolean> {
  const packageName = await getPackageName(sourceName);
  if (packageName !== "") {
    return sourceName.startsWith(`${packageName}/`);
  }
  return false;
}
