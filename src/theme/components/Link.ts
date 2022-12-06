export const Link = {
  variants: {
    'button-link-secondary': {
      color: 'primary',
      bg: 'button-bg',
      _hover: { textDecoration: 'none', bg: 'primary', color: 'bg' },
      _focus: {
        textDecoration: 'none',
        bg: 'primary',
        color: 'bg',
        boxShadow: 'inset 0 0 0 3px var(--chakra-colors-bg)'
      },
      _active: { textDecoration: 'none', bg: 'secondary', color: 'bg' }
    },
    light: {
      textDecoration: 'underline',
      color: 'primary',
      _hover: { color: 'body', textDecorationColor: 'secondary' },
      _focus: {
        color: 'primary',
        boxShadow: '0 0 0 1px var(--chakra-colors-primary) !important',
        textDecoration: 'none'
      },
      _active: {
        color: 'secondary',
        textDecorationColor: 'secondary'
      }
    }
  }
};
