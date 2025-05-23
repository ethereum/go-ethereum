import * as nodeCrypto from 'crypto';

export const crypto: { node?: any; web?: any } = {
  node: nodeCrypto,
  web: undefined,
};
