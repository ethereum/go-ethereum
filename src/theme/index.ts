import { extendTheme } from '@chakra-ui/react';

import { colors, sizes, textStyles } from './foundations';
import { Button, Link } from './components';

const overrides = {
  colors,
  components: {
    Button,
    Link
  },
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
