import { FC } from 'react';
import { Input, InputGroup, Stack } from '@chakra-ui/react'

import { BORDER_WIDTH } from '../../../constants'
import { LensIcon } from '../icons';

export const Search: FC = () => {
  return (
    <Stack
      borderBottom={{ base: BORDER_WIDTH, md: 'none' }}
      borderRight={{ base: 'none', md: BORDER_WIDTH }}
      borderColor={{ base: 'bg', md: 'primary' }}
      px={4}
      py={{ base: 8, md: 4 }}
      _hover={{ base: {bg: 'primary'}, md: {bg: 'none'}}}
    >
      <InputGroup>
        <Input
          variant='unstyled'
          placeholder='search'
          size='md'
          _placeholder={{ color: {base: 'bg', md: 'primary'}, fontStyle: 'italic' }}
        />
        <Stack pl={4} justifyContent='center' alignItems='center'>
          <LensIcon color={{ base: 'bg', md: 'primary' }} fontSize={{ base: '3xl', md: 'md' }} />
        </Stack>
      </InputGroup>
    </Stack>
  );
};