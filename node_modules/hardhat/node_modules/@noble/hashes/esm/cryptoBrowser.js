export const crypto = {
    node: undefined,
    web: typeof self === 'object' && 'crypto' in self ? self.crypto : undefined,
};
//# sourceMappingURL=cryptoBrowser.js.map