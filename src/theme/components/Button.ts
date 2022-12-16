export const Button = {
  variants: {
    primary: {
      py: '8px',
      px: '32px',
      borderRadius: 0,
      width: { base: '188px', md: 'auto' },
      bg: 'primary',
      _hover: { bg: 'secondary' },
      _focus: {
        bg: 'secondary',
        boxShadow: 'inset 0 0 0 2px var(--chakra-colors-primary) !important'
      },
      _active: {
        bg: 'secondary',
        borderTop: '4px solid',
        borderColor: 'primary',
        pt: '4px'
      }
    }
  }
};
