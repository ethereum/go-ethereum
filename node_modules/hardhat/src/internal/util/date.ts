export function parseDateString(str: string): Date {
  return new Date(str);
}

export function dateToTimestampSeconds(date: Date): number {
  return Math.floor(date.valueOf() / 1000);
}

export function timestampSecondsToDate(timestamp: number): Date {
  return new Date(timestamp * 1000);
}

export function getDifferenceInSeconds(a: Date, b: Date): number {
  return Math.floor((a.valueOf() - b.valueOf()) / 1000);
}
