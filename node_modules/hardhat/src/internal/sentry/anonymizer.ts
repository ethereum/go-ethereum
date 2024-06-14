import { Event, Exception, StackFrame, Stacktrace } from "@sentry/node";
import findup from "find-up";
import { either } from "fp-ts";
import * as path from "path";

interface WordMatch {
  index: number;
  word: string;
}

const ANONYMIZED_FILE = "<user-file>";
const ANONYMIZED_MNEMONIC = "<mnemonic>";
const MNEMONIC_PHRASE_LENGTH_THRESHOLD = 7;
const MINIMUM_AMOUNT_OF_WORDS_TO_ANONYMIZE = 4;

export class Anonymizer {
  constructor(private _configPath?: string) {}

  /**
   * Given a sentry serialized exception
   * (https://develop.sentry.dev/sdk/event-payloads/exception/), return an
   * anonymized version of the event.
   */
  public anonymize(event: any): either.Either<string, Event> {
    if (event === null || event === undefined) {
      return either.left("event is null or undefined");
    }
    if (typeof event !== "object") {
      return either.left("event is not an object");
    }

    const result: Event = {
      event_id: event.event_id,
      platform: event.platform,
      timestamp: event.timestamp,
      extra: event.extra,
    };

    if (event.exception !== undefined && event.exception.values !== undefined) {
      const anonymizededExceptions = this._anonymizeExceptions(
        event.exception.values
      );
      result.exception = {
        values: anonymizededExceptions,
      };
    }

    return either.right(result);
  }

  /**
   * Return the anonymized filename and a boolean indicating if the content of
   * the file should be anonymized
   */
  public anonymizeFilename(filename: string): {
    anonymizedFilename: string;
    anonymizeContent: boolean;
  } {
    if (filename === this._configPath) {
      const packageJsonPath = this._getFilePackageJsonPath(filename);

      if (packageJsonPath === null) {
        // if we can't find a package.json, we just return the basename
        return {
          anonymizedFilename: path.basename(filename),
          anonymizeContent: true,
        };
      }

      return {
        anonymizedFilename: path.relative(
          path.dirname(packageJsonPath),
          filename
        ),
        anonymizeContent: true,
      };
    }

    const parts = filename.split(path.sep);
    const nodeModulesIndex = parts.indexOf("node_modules");

    if (nodeModulesIndex === -1) {
      if (filename.startsWith("internal")) {
        // show internal parts of the stack trace
        return {
          anonymizedFilename: filename,
          anonymizeContent: false,
        };
      }

      // if the file isn't inside node_modules and it's a user file, we hide it completely
      return {
        anonymizedFilename: ANONYMIZED_FILE,
        anonymizeContent: true,
      };
    }

    return {
      anonymizedFilename: parts.slice(nodeModulesIndex).join(path.sep),
      anonymizeContent: false,
    };
  }

  public anonymizeErrorMessage(errorMessage: string): string {
    errorMessage = this._anonymizeMnemonic(errorMessage);

    // the \\ before path.sep is necessary for this to work on windows
    const pathRegex = new RegExp(`\\S+\\${path.sep}\\S+`, "g");

    // for files that don't have a path separator
    const fileRegex = new RegExp("\\S+\\.(js|ts)\\S*", "g");

    // hide hex strings of 20 chars or more
    const hexRegex = /(0x)?[0-9A-Fa-f]{20,}/g;

    return errorMessage
      .replace(pathRegex, ANONYMIZED_FILE)
      .replace(fileRegex, ANONYMIZED_FILE)
      .replace(hexRegex, (match) => match.replace(/./g, "x"));
  }

  public raisedByHardhat(event: Event): boolean {
    const exceptions = event?.exception?.values;

    if (exceptions === undefined) {
      // if we can't prove that the exception doesn't come from hardhat,
      // we err on the side of reporting the error
      return true;
    }

    const originalException = exceptions[exceptions.length - 1];

    const frames = originalException?.stacktrace?.frames;

    if (frames === undefined) {
      return true;
    }

    for (const frame of frames.slice().reverse()) {
      if (frame.filename === undefined) {
        continue;
      }

      // we stop after finding either a hardhat file or a file from the user's
      // project
      if (this._isHardhatFile(frame.filename)) {
        return true;
      }

      if (frame.filename === ANONYMIZED_FILE) {
        return false;
      }

      if (
        this._configPath !== undefined &&
        this._configPath.includes(frame.filename)
      ) {
        return false;
      }
    }

    // if we didn't find any hardhat frame, we don't report the error
    return false;
  }

  protected _getFilePackageJsonPath(filename: string): string | null {
    return findup.sync("package.json", {
      cwd: path.dirname(filename),
    });
  }

  private _isHardhatFile(filename: string): boolean {
    const nomiclabsPath = path.join("node_modules", "@nomiclabs");
    const nomicFoundationPath = path.join("node_modules", "@nomicfoundation");
    const truffleContractPath = path.join(nomiclabsPath, "truffle-contract");
    const isHardhatFile =
      (filename.startsWith(nomiclabsPath) ||
        filename.startsWith(nomicFoundationPath)) &&
      !filename.startsWith(truffleContractPath);

    return isHardhatFile;
  }

  private _anonymizeExceptions(exceptions: Exception[]): Exception[] {
    return exceptions.map((exception) => this._anonymizeException(exception));
  }

  private _anonymizeException(value: Exception): Exception {
    const result: Exception = {
      type: value.type,
    };

    if (value.value !== undefined) {
      result.value = this.anonymizeErrorMessage(value.value);
    }

    if (value.stacktrace !== undefined) {
      result.stacktrace = this._anonymizeStacktrace(value.stacktrace);
    }

    return result;
  }

  private _anonymizeStacktrace(stacktrace: Stacktrace): Stacktrace {
    if (stacktrace.frames !== undefined) {
      const anonymizededFrames = this._anonymizeFrames(stacktrace.frames);
      return {
        frames: anonymizededFrames,
      };
    }

    return {};
  }

  private _anonymizeFrames(frames: StackFrame[]): StackFrame[] {
    return frames.map((frame) => this._anonymizeFrame(frame));
  }

  private _anonymizeFrame(frame: StackFrame): StackFrame {
    const result: StackFrame = {
      lineno: frame.lineno,
      colno: frame.colno,
      function: frame.function,
    };

    let anonymizeContent = true;
    if (frame.filename !== undefined) {
      const anonymizationResult = this.anonymizeFilename(frame.filename);
      result.filename = anonymizationResult.anonymizedFilename;
      anonymizeContent = anonymizationResult.anonymizeContent;
    }

    if (!anonymizeContent) {
      result.context_line = frame.context_line;
      result.pre_context = frame.pre_context;
      result.post_context = frame.post_context;
      result.vars = frame.vars;
    }

    return result;
  }

  private _anonymizeMnemonic(errorMessage: string): string {
    const matches = getAllWordMatches(errorMessage);

    // If there are enough consecutive words, there's a good chance of there being a mnemonic phrase
    if (matches.length < MNEMONIC_PHRASE_LENGTH_THRESHOLD) {
      return errorMessage;
    }

    const mnemonicWordlist = ([] as string[]).concat(
      ...[
        require("ethereum-cryptography/bip39/wordlists/czech"),
        require("ethereum-cryptography/bip39/wordlists/english"),
        require("ethereum-cryptography/bip39/wordlists/french"),
        require("ethereum-cryptography/bip39/wordlists/italian"),
        require("ethereum-cryptography/bip39/wordlists/japanese"),
        require("ethereum-cryptography/bip39/wordlists/korean"),
        require("ethereum-cryptography/bip39/wordlists/simplified-chinese"),
        require("ethereum-cryptography/bip39/wordlists/spanish"),
        require("ethereum-cryptography/bip39/wordlists/traditional-chinese"),
      ].map((wordlistModule) => wordlistModule.wordlist)
    );
    let anonymizedMessage = errorMessage.slice(0, matches[0].index);

    // Determine all mnemonic phrase maximal fragments.
    // We check sequences of n consecutive words just in case there is a typo
    let wordIndex = 0;
    while (wordIndex < matches.length) {
      const maximalPhrase = getMaximalMnemonicPhrase(
        matches,
        errorMessage,
        wordIndex,
        mnemonicWordlist
      );

      if (maximalPhrase.length >= MINIMUM_AMOUNT_OF_WORDS_TO_ANONYMIZE) {
        const lastAnonymizedWord = maximalPhrase[maximalPhrase.length - 1];
        const nextWordIndex =
          wordIndex + maximalPhrase.length < matches.length
            ? matches[wordIndex + maximalPhrase.length].index
            : errorMessage.length;
        const sliceUntilNextWord = errorMessage.slice(
          lastAnonymizedWord.index + lastAnonymizedWord.word.length,
          nextWordIndex
        );
        anonymizedMessage += `${ANONYMIZED_MNEMONIC}${sliceUntilNextWord}`;
        wordIndex += maximalPhrase.length;
      } else {
        const thisWord = matches[wordIndex];
        const nextWordIndex =
          wordIndex + 1 < matches.length
            ? matches[wordIndex + 1].index
            : errorMessage.length;
        const sliceUntilNextWord = errorMessage.slice(
          thisWord.index,
          nextWordIndex
        );
        anonymizedMessage += sliceUntilNextWord;
        wordIndex++;
      }
    }

    return anonymizedMessage;
  }
}

function getMaximalMnemonicPhrase(
  matches: WordMatch[],
  originalMessage: string,
  startIndex: number,
  mnemonicWordlist: string[]
): WordMatch[] {
  const maximalPhrase: WordMatch[] = [];
  for (let i = startIndex; i < matches.length; i++) {
    const thisMatch = matches[i];
    if (!mnemonicWordlist.includes(thisMatch.word)) {
      break;
    }

    if (maximalPhrase.length > 0) {
      // Check that there's only whitespace until this word.
      const lastMatch = maximalPhrase[maximalPhrase.length - 1];
      const lastIndex = lastMatch.index + lastMatch.word.length;
      const sliceBetweenWords = originalMessage.slice(
        lastIndex,
        thisMatch.index
      );
      if (!/\s+/u.test(sliceBetweenWords)) {
        break;
      }
    }

    maximalPhrase.push(thisMatch);
  }
  return maximalPhrase;
}

function getAllWordMatches(errorMessage: string) {
  const matches: WordMatch[] = [];
  const re = /\p{Letter}+/gu;
  let match = re.exec(errorMessage);
  while (match !== null) {
    matches.push({
      word: match[0],
      index: match.index,
    });
    match = re.exec(errorMessage);
  }
  return matches;
}
