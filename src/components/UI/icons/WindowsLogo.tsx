import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [25, 24];

const Icon = createIcon({
  displayName: 'WindowsLogo',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path
        d='M0.5 12V3.354L10.5 1.999V12H0.5ZM11.5 12H24.5V0L11.5 1.807V12ZM10.5 13H0.5V20.646L10.5 22.001V13ZM11.5 13V22.194L24.5 24V13H11.5Z'
        fill='currentColor'
      />
    </svg>
  )
});

export const WindowsLogo: React.FC<IconProps> = props => <Icon h={h} w={w} color='bg' {...props} />;
