import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [25, 24];

const Icon = createIcon({
  displayName: 'WindowsLogo',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path
        d='M.5 12V3.354l10-1.355V12H.5zm11 0h13V0l-13 1.807V12zm-1 1H.5v7.646l10 1.355V13zm1 0v9.194L24.5 24V13h-13z'
        fill='currentColor'
      />
    </svg>
  )
});

export const WindowsLogo: React.FC<IconProps> = props => <Icon h={h} w={w} color='bg' {...props} />;
