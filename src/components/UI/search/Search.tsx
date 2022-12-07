import { FC, useState } from 'react';
import { Input, InputGroup, Link, Stack } from '@chakra-ui/react';

import { BORDER_WIDTH } from '../../../constants';
import { LensIcon } from '../icons';

export const Search: FC = () => {
  const [query, setQuery] = useState<string>('');

  // Handlers
  const handleSubmit = (e: React.FormEvent<HTMLInputElement>): void => {
    document.getElementById('search-link')?.click();
  };
  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === 'Enter') handleSubmit(e);
  };
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
      <InputGroup>
        <Input
          py={{ base: 8, md: 4 }}
          px={4}
          variant='unstyled'
          placeholder='search'
          size='md'
          _placeholder={{ color: { base: 'bg', md: 'primary' }, fontStyle: 'italic' }}
          value={query}
          onChange={handleChange}
          onKeyDown={handleKeyPress}
          outlineOffset={4}
        />
        <Link
          href={`https://www.google.com/search?q=site:geth.ethereum.org%20${encodeURIComponent(query)}`}
          isExternal
          display='grid'
          placeItems='center'
          id="search-link"
          px={4}
          py={{ base: 8, md: 4 }}
          _focusVisible={{ outline: '2px solid var(--chakra-colors-primary)', outlineOffset: -4 }}
        >
          <LensIcon color={{ base: 'bg', md: 'primary' }} fontSize={{ base: '3xl', md: 'lg' }} />
        </Link>
      </InputGroup>
    </Stack>
  );
};
