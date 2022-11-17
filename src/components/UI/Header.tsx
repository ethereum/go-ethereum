import { Box, Flex, Input, InputGroup, Link, Stack, Text, useColorMode } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { HamburgerIcon, LensIcon, MoonIcon, SunIcon } from '../UI/icons';
import { DOCS_PAGE, DOWNLOADS_PAGE } from '../../constants';

export const Header: FC = () => {
  const { colorMode, toggleColorMode } = useColorMode();
  const isDark = colorMode === 'dark';
  return (
    <Flex mb={4} border='2px solid' borderColor='primary' justifyContent='space-between'>
      <Stack
        p={4}
        justifyContent='center'
        alignItems='flex-start'
        borderRight='2px'
        borderColor='primary'
        flexGrow={2}
      >
        <NextLink href={'/'} passHref>
          <Link _hover={{ textDecoration: 'none' }}>
            <Text textStyle='header-font'>go-ethereum</Text>
          </Link>
        </NextLink>
      </Stack>

      <Flex>
        {/* DOWNLOADS */}
        <Stack
          p={4}
          justifyContent='center'
          borderRight='2px'
          borderColor='primary'
          display={{ base: 'none', md: 'block' }}
          color='primary'
          _hover={{
            textDecoration: 'none',
            bg: 'primary',
            color: 'bg !important'
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
          borderRight='2px'
          borderColor='primary'
          display={{ base: 'none', md: 'block' }}
          color='primary'
          _hover={{
            textDecoration: 'none',
            bg: 'primary',
            color: 'bg !important'
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
          borderRight='2px'
          borderColor='primary'
        >
          <InputGroup>
            <Input
              variant='unstyled'
              placeholder='search'
              size='md'
              _placeholder={{ color: 'primary', fontStyle: 'italic' }}
            />

            <Stack pl={4} justifyContent='center' alignItems='center'>
              <LensIcon color='primary' />
            </Stack>
          </InputGroup>
        </Stack>

        {/* DARK MODE SWITCH */}
        <Box
          as='button'
          p={4}
          borderRight={{ base: '2px', md: 'none' }}
          borderColor='primary'
          onClick={toggleColorMode}
          _hover={{
            bg: 'primary',
            svg: { color: 'bg' }
          }}
        >
          {isDark ? <SunIcon color='primary' /> : <MoonIcon color='primary' />}
        </Box>

        {/* HAMBURGER MENU */}
        <Box p={4} display={{ base: 'block', md: 'none' }}>
          <HamburgerIcon color='primary' />
        </Box>
      </Flex>
    </Flex>
  );
};
