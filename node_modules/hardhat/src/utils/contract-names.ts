import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";

/**
 * Returns a fully qualified name from a sourceName and contractName.
 */
export function getFullyQualifiedName(
  sourceName: string,
  contractName: string
): string {
  return `${sourceName}:${contractName}`;
}

/**
 * Returns true if a name is fully qualified, and not just a bare contract name.
 */
export function isFullyQualifiedName(name: string): boolean {
  return name.includes(":");
}

/**
 * Parses a fully qualified name.
 *
 * @param fullyQualifiedName It MUST be a fully qualified name.
 * @throws {HardhatError} If the name is not fully qualified.
 */
export function parseFullyQualifiedName(fullyQualifiedName: string): {
  sourceName: string;
  contractName: string;
} {
  const { sourceName, contractName } = parseName(fullyQualifiedName);

  if (sourceName === undefined) {
    throw new HardhatError(ERRORS.CONTRACT_NAMES.INVALID_FULLY_QUALIFIED_NAME, {
      name: fullyQualifiedName,
    });
  }

  return { sourceName, contractName };
}

/**
 * Parses a name, which can be a bare contract name, or a fully qualified name.
 */
export function parseName(name: string): {
  sourceName?: string;
  contractName: string;
} {
  const parts = name.split(":");

  if (parts.length === 1) {
    return { contractName: parts[0] };
  }

  const contractName = parts[parts.length - 1];
  const sourceName = parts.slice(0, parts.length - 1).join(":");

  return { sourceName, contractName };
}

/**
 * Returns the edit-distance between two given strings using Levenshtein distance.
 *
 * @param a First string being compared
 * @param b Second string being compared
 * @returns distance between the two strings (lower number == more similar)
 * @see https://github.com/gustf/js-levenshtein
 * @license MIT - https://github.com/gustf/js-levenshtein/blob/master/LICENSE
 */
export function findDistance(a: string, b: string): number {
  function _min(
    _d0: number,
    _d1: number,
    _d2: number,
    _bx: number,
    _ay: number
  ): number {
    return _d0 < _d1 || _d2 < _d1
      ? _d0 > _d2
        ? _d2 + 1
        : _d0 + 1
      : _bx === _ay
      ? _d1
      : _d1 + 1;
  }

  if (a === b) {
    return 0;
  }

  if (a.length > b.length) {
    [a, b] = [b, a];
  }

  let la = a.length;
  let lb = b.length;

  while (la > 0 && a.charCodeAt(la - 1) === b.charCodeAt(lb - 1)) {
    la--;
    lb--;
  }

  let offset = 0;

  while (offset < la && a.charCodeAt(offset) === b.charCodeAt(offset)) {
    offset++;
  }

  la -= offset;
  lb -= offset;

  if (la === 0 || lb < 3) {
    return lb;
  }

  let x = 0;
  let y: number;
  let d0: number;
  let d1: number;
  let d2: number;
  let d3: number;
  let dd: number = 0; // typescript gets angry if we don't assign here
  let dy: number;
  let ay: number;
  let bx0: number;
  let bx1: number;
  let bx2: number;
  let bx3: number;

  const vector = [];

  for (y = 0; y < la; y++) {
    vector.push(y + 1);
    vector.push(a.charCodeAt(offset + y));
  }

  const len = vector.length - 1;

  for (; x < lb - 3; ) {
    bx0 = b.charCodeAt(offset + (d0 = x));
    bx1 = b.charCodeAt(offset + (d1 = x + 1));
    bx2 = b.charCodeAt(offset + (d2 = x + 2));
    bx3 = b.charCodeAt(offset + (d3 = x + 3));
    dd = x += 4;
    for (y = 0; y < len; y += 2) {
      dy = vector[y];
      ay = vector[y + 1];
      d0 = _min(dy, d0, d1, bx0, ay);
      d1 = _min(d0, d1, d2, bx1, ay);
      d2 = _min(d1, d2, d3, bx2, ay);
      dd = _min(d2, d3, dd, bx3, ay);
      vector[y] = dd;
      d3 = d2;
      d2 = d1;
      d1 = d0;
      d0 = dy;
    }
  }

  for (; x < lb; ) {
    bx0 = b.charCodeAt(offset + (d0 = x));
    dd = ++x;
    for (y = 0; y < len; y += 2) {
      dy = vector[y];
      vector[y] = dd = _min(dy, d0, dd, bx0, vector[y + 1]);
      d0 = dy;
    }
  }

  return dd;
}
