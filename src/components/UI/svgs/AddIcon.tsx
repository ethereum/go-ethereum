import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [24, 24];

const Icon = createIcon({
  displayName: 'AddIcon',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill='none' xmlns='http://www.w3.org/2000/svg'>
      <g fill='currentColor'>
        <path d='M2 11h20v2H2z' />
        <path d='M11 2h2v20h-2z' />
      </g>
    </svg>
  )
});

export const AddIcon: React.FC<IconProps> = props => (
  <Icon h={h} w={w} color='primary' {...props} />
);
