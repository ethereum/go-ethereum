import { extendTheme } from '@chakra-ui/react';

import { colors, shadows, sizes, textStyles } from './foundations';
import { Button, Link } from './components';

const overrides = {
  colors,
  components: {
    Button,
    Link
  },
  shadows,
  sizes,
  styles: {
    global: () => ({
      body: {
        bg: 'yellow.50'
      }
    })
  },
  textStyles
};

export default extendTheme(overrides);
