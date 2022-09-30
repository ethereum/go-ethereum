import { extendTheme } from '@chakra-ui/react';

import { colors, sizes } from './foundations';

const overrides = {
  colors,
  components: {},
  sizes,
  styles: {
    global: () => ({
      body: {
        // TODO: move color to theme colors
        bg: '#f0f2e2'
      }
    })
  },
  // TODO: fix textStyles
  textStyles: {
    h1: {},
    h2: {},
    'hero-text-small': {
      fontSize: '13px',
      fontFamily: '"Inter", sans-serif'
    },
    // TODO: refactor w/ semantic tokens for light/dark mode
    'link-light': {},
    // TODO: refactor w/ semantic tokens for light/dark mode
    'text-light': {}
  }
};

export default extendTheme(overrides);
