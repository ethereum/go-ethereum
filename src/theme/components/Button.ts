export const Button = {
  variants: {
    primary: {
      py: '8px',
      px: '32px',
      borderRadius: 0,
      width: { base: '188px', md: 'auto' },
      // TODO: move to theme colors
      bg: 'brand.light.primary',
      _hover: { bg: 'brand.light.secondary' },
      _focus: {
        bg: 'brand.light.primary',
        boxShadow: 'inset 0 0 0 2px #06fece !important'
      },
      _active: { borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }
    }
  }
};
