import ansiEscapes from "ansi-escapes";

export function printLine(line: string) {
  console.log(line);
}

export function replaceLastLine(newLine: string) {
  if (process.stdout.isTTY === true) {
    process.stdout.write(
      // eslint-disable-next-line prefer-template
      ansiEscapes.cursorHide +
        ansiEscapes.cursorPrevLine +
        newLine +
        ansiEscapes.eraseEndLine +
        "\n" +
        ansiEscapes.cursorShow
    );
  } else {
    process.stdout.write(`${newLine}\n`);
  }
}

export interface LoggerConfig {
  enabled: boolean;
  printLineFn?: (line: string) => void;
  replaceLastLineFn?: (line: string) => void;
}
