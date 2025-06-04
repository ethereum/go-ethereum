// Global symbol available in browsers only
declare const self: Record<string, any> | undefined;
export const crypto: { node?: any; web?: any } = {
  node: undefined,
  web: typeof self === 'object' && 'crypto' in self ? self.crypto : undefined,
};
