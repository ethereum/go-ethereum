import { LiteralUnion } from 'type-fest';

export interface StackFrame {
  file: string | null;
  methodName: LiteralUnion<'<unknown>', string>;
  arguments: string[];
  lineNumber: number | null;
  column: number | null;
}

/**
 * This parser parses a stack trace from any browser or Node.js and returns an array of hashes representing a line.
 * 
 * @param stackString - The stack to parse, usually from `error.stack` property.
 * @returns The parsed stack frames.
 */
export function parse(stackString: string): StackFrame[];
