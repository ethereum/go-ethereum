import { FC } from 'react';
import { Stack } from '@chakra-ui/react';

import { DocsLinks } from './DocsLinks';

import { NavLink } from '../../../types';

interface Props {
  navLinks: NavLink[];
}

export const DocsNav: FC<Props> = ({ navLinks }) => {
  return (
    <Stack w={{ base: '100%', lg: 72 }} as='aside' display={{ base: 'none', lg: 'block' }}>
      <DocsLinks navLinks={navLinks} />
    </Stack>
  );
};
