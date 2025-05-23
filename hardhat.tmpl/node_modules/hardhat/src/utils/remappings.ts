export function applyRemappings(
  remappings: Record<string, string>,
  sourceName: string
): string {
  const selectedRemapping = { from: "", to: "" };

  for (const [from, to] of Object.entries(remappings)) {
    if (
      sourceName.startsWith(from) &&
      from.length >= selectedRemapping.from.length
    ) {
      [selectedRemapping.from, selectedRemapping.to] = [from, to];
    }
  }

  return sourceName.replace(selectedRemapping.from, selectedRemapping.to);
}
