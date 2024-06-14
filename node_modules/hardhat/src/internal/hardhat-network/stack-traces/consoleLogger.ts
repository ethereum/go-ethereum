import {
  bytesToBigInt,
  bytesToHex as bufferToHex,
  bytesToInt,
  fromSigned,
} from "@nomicfoundation/ethereumjs-util";
import util from "util";

import {
  AddressTy,
  BoolTy,
  Bytes10Ty,
  Bytes11Ty,
  Bytes12Ty,
  Bytes13Ty,
  Bytes14Ty,
  Bytes15Ty,
  Bytes16Ty,
  Bytes17Ty,
  Bytes18Ty,
  Bytes19Ty,
  Bytes1Ty,
  Bytes20Ty,
  Bytes21Ty,
  Bytes22Ty,
  Bytes23Ty,
  Bytes24Ty,
  Bytes25Ty,
  Bytes26Ty,
  Bytes27Ty,
  Bytes28Ty,
  Bytes29Ty,
  Bytes2Ty,
  Bytes30Ty,
  Bytes31Ty,
  Bytes32Ty,
  Bytes3Ty,
  Bytes4Ty,
  Bytes5Ty,
  Bytes6Ty,
  Bytes7Ty,
  Bytes8Ty,
  Bytes9Ty,
  BytesTy,
  ConsoleLogs,
  Int256Ty,
  StringTy,
  Uint256Ty,
} from "./logger";
import {
  EvmMessageTrace,
  isCallTrace,
  isEvmStep,
  isPrecompileTrace,
  MessageTrace,
} from "./message-trace";

const CONSOLE_ADDRESS = "0x000000000000000000636F6e736F6c652e6c6f67"; // toHex("console.log")
const REGISTER_SIZE = 32;

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface ConsoleLogArray extends Array<ConsoleLogEntry> {}

export type ConsoleLogEntry = string | ConsoleLogArray;

// eslint-disable-next-line @typescript-eslint/no-redeclare
export type ConsoleLogs = ConsoleLogEntry[];

export class ConsoleLogger {
  private readonly _consoleLogs: {
    [key: number]: string[];
  } = {};

  constructor() {
    this._consoleLogs = ConsoleLogs;
  }

  public getLogMessages(maybeDecodedMessageTrace: MessageTrace): string[] {
    return this.getExecutionLogs(maybeDecodedMessageTrace).map(
      consoleLogToString
    );
  }

  public getExecutionLogs(
    maybeDecodedMessageTrace: MessageTrace
  ): ConsoleLogs[] {
    if (isPrecompileTrace(maybeDecodedMessageTrace)) {
      return [];
    }

    const logs: ConsoleLogs[] = [];
    this._collectExecutionLogs(maybeDecodedMessageTrace, logs);
    return logs;
  }

  private _collectExecutionLogs(trace: EvmMessageTrace, logs: ConsoleLogs) {
    for (const messageTrace of trace.steps) {
      if (isEvmStep(messageTrace) || isPrecompileTrace(messageTrace)) {
        continue;
      }

      if (
        isCallTrace(messageTrace) &&
        bufferToHex(messageTrace.address) === CONSOLE_ADDRESS.toLowerCase()
      ) {
        const log = this._maybeConsoleLog(Buffer.from(messageTrace.calldata));
        if (log !== undefined) {
          logs.push(log);
        }

        continue;
      }

      this._collectExecutionLogs(messageTrace, logs);
    }
  }

  /**
   * Temporary code to print console.sol messages that come from EDR
   */
  public getDecodedLogs(messages: Buffer[]): string[] {
    const logs: string[] = [];

    for (const message of messages) {
      const log = this._maybeConsoleLog(message);
      if (log !== undefined) {
        logs.push(consoleLogToString(log));
      }
    }

    return logs;
  }

  private _maybeConsoleLog(calldata: Buffer): ConsoleLogs | undefined {
    const sig = bytesToInt(calldata.slice(0, 4));
    const parameters = calldata.slice(4);

    const types = this._consoleLogs[sig];
    if (types === undefined) {
      return;
    }

    const consoleLogs = this._decode(parameters, types);

    this._replaceNumberFormatSpecifiers(consoleLogs);

    return consoleLogs;
  }

  private _replaceNumberFormatSpecifiers(consoleLogs: ConsoleLogs) {
    /**
     * Replace the occurrences of %d and %i with %s. This is necessary because if the arguments passed are numbers,
     * they could be too large to be formatted as a Number or an Integer, so it is safer to use a String.
     * %d and %i are replaced only if there is an odd number of % before the d or i.
     * If there is an even number of % then it is assumed that the % is escaped and should not be replaced.
     * The regex matches a '%d' or an '%i' that has an even number of
     * '%' behind it (including 0). This group of pairs of '%' is captured
     * and preserved, while the '%[di]' is replaced with '%s'.
     * Naively doing (%%)* is not enough; we also have to use the
     * (?<!%) negative look-behind to make this work.
     * The (?:) is just to avoid capturing that inner group.
     */
    if (consoleLogs.length > 0 && typeof consoleLogs[0] === "string") {
      consoleLogs[0] = consoleLogs[0].replace(
        /((?<!%)(?:%%)*)(%[di])/g,
        "$1%s"
      );
    }
  }

  private _decode(data: Buffer, types: string[]): ConsoleLogs {
    return types.map((type, i) => {
      const position: number = i * 32;
      switch (types[i]) {
        case Uint256Ty:
          return bytesToBigInt(
            data.slice(position, position + REGISTER_SIZE)
          ).toString(10);

        case Int256Ty:
          return fromSigned(
            data.slice(position, position + REGISTER_SIZE)
          ).toString();

        case BoolTy:
          if (data[position + 31] !== 0) {
            return "true";
          }
          return "false";

        case StringTy:
          const sStart = bytesToInt(
            data.slice(position, position + REGISTER_SIZE)
          );
          const sLen = bytesToInt(data.slice(sStart, sStart + REGISTER_SIZE));
          return data
            .slice(sStart + REGISTER_SIZE, sStart + REGISTER_SIZE + sLen)
            .toString();

        case AddressTy:
          return bufferToHex(
            data.slice(position + 12, position + REGISTER_SIZE)
          );

        case BytesTy:
          const bStart = bytesToInt(
            data.slice(position, position + REGISTER_SIZE)
          );
          const bLen = bytesToInt(data.slice(bStart, bStart + REGISTER_SIZE));
          return bufferToHex(
            data.slice(bStart + REGISTER_SIZE, bStart + REGISTER_SIZE + bLen)
          );

        case Bytes1Ty:
          return bufferToHex(data.slice(position, position + 1));
        case Bytes2Ty:
          return bufferToHex(data.slice(position, position + 2));
        case Bytes3Ty:
          return bufferToHex(data.slice(position, position + 3));
        case Bytes4Ty:
          return bufferToHex(data.slice(position, position + 4));
        case Bytes5Ty:
          return bufferToHex(data.slice(position, position + 5));
        case Bytes6Ty:
          return bufferToHex(data.slice(position, position + 6));
        case Bytes7Ty:
          return bufferToHex(data.slice(position, position + 7));
        case Bytes8Ty:
          return bufferToHex(data.slice(position, position + 8));
        case Bytes9Ty:
          return bufferToHex(data.slice(position, position + 9));
        case Bytes10Ty:
          return bufferToHex(data.slice(position, position + 10));
        case Bytes11Ty:
          return bufferToHex(data.slice(position, position + 11));
        case Bytes12Ty:
          return bufferToHex(data.slice(position, position + 12));
        case Bytes13Ty:
          return bufferToHex(data.slice(position, position + 13));
        case Bytes14Ty:
          return bufferToHex(data.slice(position, position + 14));
        case Bytes15Ty:
          return bufferToHex(data.slice(position, position + 15));
        case Bytes16Ty:
          return bufferToHex(data.slice(position, position + 16));
        case Bytes17Ty:
          return bufferToHex(data.slice(position, position + 17));
        case Bytes18Ty:
          return bufferToHex(data.slice(position, position + 18));
        case Bytes19Ty:
          return bufferToHex(data.slice(position, position + 19));
        case Bytes20Ty:
          return bufferToHex(data.slice(position, position + 20));
        case Bytes21Ty:
          return bufferToHex(data.slice(position, position + 21));
        case Bytes22Ty:
          return bufferToHex(data.slice(position, position + 22));
        case Bytes23Ty:
          return bufferToHex(data.slice(position, position + 23));
        case Bytes24Ty:
          return bufferToHex(data.slice(position, position + 24));
        case Bytes25Ty:
          return bufferToHex(data.slice(position, position + 25));
        case Bytes26Ty:
          return bufferToHex(data.slice(position, position + 26));
        case Bytes27Ty:
          return bufferToHex(data.slice(position, position + 27));
        case Bytes28Ty:
          return bufferToHex(data.slice(position, position + 28));
        case Bytes29Ty:
          return bufferToHex(data.slice(position, position + 29));
        case Bytes30Ty:
          return bufferToHex(data.slice(position, position + 30));
        case Bytes31Ty:
          return bufferToHex(data.slice(position, position + 31));
        case Bytes32Ty:
          return bufferToHex(data.slice(position, position + 32));

        default:
          return "";
      }
    });
  }
}

export function consoleLogToString(log: ConsoleLogs): string {
  if (log === undefined) {
    return "";
  }

  // special case for console.log()
  if (log.length === 0) {
    return "";
  }

  return util.format(log[0], ...log.slice(1));
}
