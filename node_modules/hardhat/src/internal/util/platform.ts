export function getRequireCachedFiles(): string[] {
  return Object.keys(require.cache).filter(
    (p) => !p.startsWith("internal") && (p.endsWith(".js") || p.endsWith(".ts"))
  );
}
