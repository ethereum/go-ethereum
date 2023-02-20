import { createIcon } from '@chakra-ui/icons';

const [w, h] = [24, 24];

export const MinusIcon = createIcon({
  displayName: 'MinusIcon',
  viewBox: `0 0 ${w} ${h}`,
  d: 'M2 11h20v2H2z',
  defaultProps: {
    color: 'primary'
  }
});
