export const Link = {
  variants: {
    'button-link-secondary': {
      color: 'brand.light.primary',
      bg: 'green.50',
      _hover: { textDecoration: 'none', bg: 'brand.light.primary', color: 'yellow.50' },
      _focus: {
        textDecoration: 'none',
        bg: 'brand.light.primary',
        color: 'yellow.50',
        boxShadow: 'inset 0 0 0 3px #f0f2e2 !important'
      },
      _active: { textDecoration: 'none', bg: 'brand.light.secondary', color: 'yellow.50' }
    },
    href: {
      color: 'brand.light.primary',
      _hover: {
        color: 'brand.light.body',
        textDecorationColor: 'brand.light.body' 
      },
      _focus: {
        color: 'brand.light.primary',
        boxShadow: 'linkBoxShadow',
        textDecoration: 'none'
      },
      _pressed: { 
        color: 'brand.light.secondary',
        textDecorationColor: 'brand.light.secondary'
      }
    },
    light: {
      textDecoration: 'underline',
      color: 'brand.light.primary',
      _hover: { color: 'brand.light.body', textDecorationColor: 'brand.light.body' },
      _focus: {
        color: 'brand.light.primary',
        boxShadow: '0 0 0 1px #11866f !important',
        textDecoration: 'none'
      },
      _pressed: {
        color: 'brand.light.secondary',
        textDecorationColor: 'brand.light.secondary'
      }
    },
  }
};
