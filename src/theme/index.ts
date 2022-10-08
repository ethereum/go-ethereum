import { extendTheme } from '@chakra-ui/react';

import { colors, sizes } from './foundations';
import { Button } from './components';

const overrides = {
  colors,
  components: {
    Button
  },
  sizes,
  styles: {
    global: () => ({
      body: {
        bg: 'yellow.50'
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
