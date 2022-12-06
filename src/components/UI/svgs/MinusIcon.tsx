import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [24, 24];

const Icon = createIcon({
  displayName: 'MinusIcon',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill='none' xmlns='http://www.w3.org/2000/svg'>
      <g fill="currentColor">
        <rect height="2" width="20" x="2" y="11"></rect>
      </g>
    </svg>
  )
});

export const MinusIcon: React.FC<IconProps> = props => (
  <Icon h={h} w={w} color='primary' {...props} />
);
