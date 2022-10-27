import { Box, Flex, Input, InputGroup, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { HamburguerIcon, LensIcon, MoonIcon } from '../UI/icons';
import { DOCS_PAGE, DOWNLOADS_PAGE } from '../../constants';

export const Header: FC = () => {
  return (
    <Flex
      mb={4}
      border='2px solid'
      borderColor='brand.light.primary'
      justifyContent='space-between'
    >
      <Stack
        p={4}
        justifyContent='center'
        alignItems='flex-start'
        borderRight={{ base: 'none', sm: '2px solid #11866f'}}
        flexGrow={2}
      >
        <Text textStyle='header-font'>
          go-ethereum
        </Text>
      </Stack>

      <Flex>
        {/* DOWNLOADS */}
        <Stack
          p={4}
          justifyContent='center'
          borderRight='2px solid'
          borderColor='brand.light.primary'
          display={{ base: 'none', md: 'block' }}
          color='brand.light.primary'
          _hover={{
            textDecoration: 'none',
            bg: 'brand.light.primary',
            color: 'yellow.50 !important'
          }}
        >
          <NextLink href={DOWNLOADS_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Text textStyle='header-font' textTransform='uppercase'>
                downloads
              </Text>
            </Link>
          </NextLink>
        </Stack>

        {/* DOCUMENTATION */}
        <Stack
          p={4}
          justifyContent='center'
          borderRight={{ base: 'none', md: '2px solid #11866f' }}
          display={{ base: 'none', md: 'block' }}
          color='brand.light.primary'
          _hover={{
            textDecoration: 'none',
            bg: 'brand.light.primary',
            color: 'yellow.50 !important'
          }}
        >
          <NextLink href={DOCS_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Text textStyle='header-font' textTransform='uppercase'>
                documentation
              </Text>
            </Link>
          </NextLink>
        </Stack>

        {/* SEARCH */}
        <Stack
          p={4}
          display={{ base: 'none', md: 'block' }}
          borderRight={{ base: 'none', md: '2px solid #11866f' }}
        >
          <InputGroup>
            <Input
              variant='unstyled'
              placeholder='search'
              size='md'
              _placeholder={{ color: 'brand.light.primary', fontStyle: 'italic' }}
            />

            <Stack pl={4} justifyContent='center' alignItems='center'>
              <LensIcon />
            </Stack>
          </InputGroup>
        </Stack>

        {/* DARK MODE SWITCH */}
        <Box
          p={4}
          borderRight={{ base: '2px solid #11866f', md: 'none' }}
        >
          <MoonIcon />
        </Box>

        {/* HAMBURGUER MENU */}
        <Box p={4} display={{ base: 'block', md: 'none' }}>
          <HamburguerIcon />
        </Box>
      </Flex>
    </Flex>
  );
};
