export const textStyles = {
  h1: {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 700,
    fontSize: '2.75rem',
    lineHeight: '3.375rem',
    letterSpacing: '5%',
    color: 'body'
  },
  h2: {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 400,
    fontSize: '1.5rem',
    lineHeight: 'auto',
    letterSpacing: '4%',
    color: 'body'
  },
  'header-font': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 700,
    fontSize: { base: '0.86rem', sm: '1rem' }
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
    color: 'bg',
    fontWeight: 700,
    textTransform: 'uppercase'
  },
  'home-section-link-label': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 700,
    textTransform: 'uppercase',
    textAlign: 'center',
    p: 4
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
    color: 'primary',
    _groupHover: { color: 'bg' },
    _groupActive: { color: 'bg' },
    _groupFocus: { color: 'bg' }
  },
  'hero-text-small': {
    fontSize: '13px',
    fontFamily: '"Inter", sans-serif'
  },
  'footer-link-label': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 700,
    textTransform: 'uppercase',
    lineHeight: '21.12px',
    letterSpacing: '2%'
  },
  'footer-text': {
    fontFamily: '"Inter", sans-serif',
    lineHeight: '22px',
    fontWeight: 400,
    fontSize: '12px'
  },
  'downloads-button-label': {
    fontFamily: '"JetBrains Mono", monospace',
    color: 'bg',
    fontSize: { base: 'md', lg: 'xl' },
    textTransform: 'uppercase'
  },
  'downloads-button-sublabel': {
    fontFamily: '"JetBrains Mono", monospace',
    color: 'bg',
    fontSize: { base: 'xs', lg: 'sm' },
    textTransform: 'uppercase'
  },
  'download-tab-label': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 700,
    textTransform: 'uppercase',
    textAlign: 'center',
    fontSize: 'sm'
  },
  'inline-code-snippet': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 400,
    fontSize: 'md',
    lineHeight: 4,
    letterSpacing: '1%'
  },
  'code-block': {
    fontFamily: '"JetBrains Mono", monospace',
    fontWeight: 400,
    fontSize: 'md',
    lineHeight: '21.12px',
    letterSpacing: '1%'
  },
  // TODO: refactor w/ semantic tokens for light/dark mode
  'link-light': {},
  // TODO: refactor w/ semantic tokens for light/dark mode
  'text-light': {}
};
