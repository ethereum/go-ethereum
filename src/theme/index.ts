import { extendTheme } from '@chakra-ui/react';

import { config, colors, fonts, shadows, sizes, textStyles } from './foundations';
import { Button, Link } from './components';

const overrides = {
  config,
  colors,
  components: {
    Button,
    Link
  },
  fonts,
  shadows,
  sizes,
  styles: {
    global: () => ({
      body: {
        bg: 'bg',
        transition: 'all 200ms linear !important'
      },
      code: {
        fontFamily: 'code !important'
      }
    })
  },
  textStyles,
  semanticTokens: {
    colors: {
      primary: { _light: 'green.700', _dark: 'green.200' },
      secondary: { _light: 'green.800', _dark: 'green.600' },
      'button-bg': { _light: 'green.50', _dark: 'green.900' },
      body: { _light: 'gray.800', _dark: 'yellow.50' },
      'code-bg': { _light: 'gray.200', _dark: 'gray.900' },
      'terminal-bg': { _light: 'gray.800', _dark: 'gray.900' },
      'terminal-text': { _light: 'green.50', _dark: 'green.200' },
      bg: { _light: 'yellow.50', _dark: 'gray.800' }
    }
  }
};

export default extendTheme(overrides);
