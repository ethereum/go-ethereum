import { extendTheme } from '@chakra-ui/react';

import { colors, shadows, sizes } from './foundations';
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
  textStyles: {
    h1: {
      fontFamily: '"JetBrains Mono", monospace',
      fontWeight: 700,
      fontSize: '2.75rem',
      lineHeight: '3.375rem',
      letterSpacing: '5%',
      color: 'brand.light.body'
    },
    h2: {
      fontFamily: '"JetBrains Mono", monospace',
      fontWeight: 400,
      fontSize: '1.5rem',
      lineHeight: 'auto',
      letterSpacing: '4%',
      color: 'brand.light.body'
    },
    'homepage-description': {
      fontFamily: '"JetBrains Mono", monospace',
      fontWeight: 700,
      lineHeight: '21px',
      letterSpacing: '0.05em',
      textAlign: { base: 'center', md: 'left' }
    },
    'homepage-primary-label': {
      fontFamily: '"JetBrains Mono", monospace',
      color: 'yellow.50',
      fontWeight: 700,
      textTransform: 'uppercase'
    },
    'quick-link-text': {
      fontFamily: '"Inter", sans-serif',
      lineHeight: '26px'
    },
    'quick-link-label': {
      fontFamily: '"JetBrains Mono", monospace',
      fontWeight: 700,
      textTransform: 'uppercase',
      textAlign: 'center',
      color: 'brand.light.primary',
      _groupHover: { color: 'yellow.50' },
      _groupActive: { color: 'yellow.50' },
      _groupFocus: { color: 'yellow.50' }
    },
    'hero-text-small': {
      fontSize: '13px',
      fontFamily: '"Inter", sans-serif'
    },
    'downloads-button-label': {
      fontFamily:'"JetBrains Mono", monospace',
      color:'yellow.50',
      fontSize:'xs',
      textTransform:'uppercase',
    },
    'download-tab-label': {
      fontFamily: '"JetBrains Mono", monospace',
      fontWeight: 700,
      textTransform: 'uppercase',
      textAlign: 'center',
      fontSize: 'sm',
    },
    // TODO: refactor w/ semantic tokens for light/dark mode
    'link-light': {},
    // TODO: refactor w/ semantic tokens for light/dark mode
    'text-light': {}
  }
};

export default extendTheme(overrides);
