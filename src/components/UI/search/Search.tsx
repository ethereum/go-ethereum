import { FC, useState } from 'react';
import { Button, Input, InputGroup, Stack } from '@chakra-ui/react';

import { BORDER_WIDTH } from '../../../constants';
import { LensIcon } from '../icons';

export const Search: FC = () => {
  const [query, setQuery] = useState<string>('');

  // Handlers
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    setQuery(e.target.value);
  };

  return (
    <Stack
      borderBottom={{ base: BORDER_WIDTH, md: 'none' }}
      borderRight={{ base: 'none', md: BORDER_WIDTH }}
      borderColor={{ base: 'bg', md: 'primary' }}
      _hover={{ base: { bg: 'primary' }, md: { bg: 'none' } }}
    >
      <form method='get' action='https://duckduckgo.com/' role='search' target='blank'>
        <InputGroup alignItems='center'>
          <Input type="hidden" name="sites" value="geth.ethereum.org" />
          <Input
            type="text"
            name="q"
            py={{ base: 8, md: 4 }}
            px={4}
            variant='unstyled'
            placeholder='search'
            size='md'
            _placeholder={{ color: { base: 'bg', md: 'primary' }, fontStyle: 'italic' }}
            value={query}
            onChange={handleChange}
            outlineOffset={4}
          />
          <Button
            type="submit"
            px={4}
            me={2}
            borderRadius='0'
            bg='none'
            _focusVisible={{
              outline: '2px solid var(--chakra-colors-primary)',
              outlineOffset: -2
            }}
            _hover={{
              bg: 'primary',
              svg: { color: 'bg' }
            }}
          >
            <LensIcon color={{ base: 'bg', md: 'primary' }} fontSize={{ base: '3xl', md: 'xl' }} />
          </Button>
        </InputGroup>
      </form>
    </Stack>
  );
};
