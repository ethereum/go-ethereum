import { createIcon } from '@chakra-ui/icons';

const [w, h] = [180, 278];

export const GlyphHome = createIcon({
  displayName: 'GlyphHome',
  viewBox: `0 0 ${w} ${h}`,
  d: 'M90 276.5v-69.121L2.765 157.376 90 276.5zM90 276.5v-69.121l87.236-50.003L90 276.5zM90 190.325v-87.442L1.5 141.27 90 190.325zM90 190.325v-87.442l88.5 38.387L90 190.325zM1.5 140.901 90 1.5v100.76L1.5 140.901zM178.5 140.901 90 1.5v100.76l88.5 38.641z',
  defaultProps: {
    color: 'primary',
    w,
    h
  }
});
