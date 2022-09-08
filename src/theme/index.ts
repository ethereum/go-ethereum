import { extendTheme } from '@chakra-ui/react';

import { breakpoints, sizes } from './foundations';

const overrides = {
  breakpoints,
  sizes
};

export default extendTheme(overrides);
