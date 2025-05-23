export function getCurrentTimestamp(): number {
  return Math.ceil(new Date().getTime() / 1000);
}
