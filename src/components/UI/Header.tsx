import { FC } from 'react';
import { Box, Flex, Link, Stack, Text, useColorMode } from '@chakra-ui/react';
import NextLink from 'next/link';

import { HeaderButtons, Search } from './';
import { MoonIcon, SunIcon } from '../UI/icons';

import { MobileMenu } from '../layouts';

export const Header: FC = () => {
  const { colorMode, toggleColorMode } = useColorMode();
  const isDark = colorMode === 'dark';

  return (
    <Flex
      mb={{ base: 4, lg: 8 }}
      border='2px'
      borderColor='primary'
      justifyContent='space-between'
      position='relative'
    >
      <Flex
        p={4}
        justifyContent='flex-start'
        alignItems='center'
        borderRight='2px'
        borderColor='primary'
        flex={1}
        gap={6}
      >
        <NextLink href={'/'} passHref legacyBehavior>
          <Link _hover={{ textDecoration: 'none' }}>
            <Text textStyle='header-font' whiteSpace='nowrap'>
              go-ethereum
            </Text>
          </Link>
        </NextLink>
        <Box
          as='a'
          href='#main-content'
          pointerEvents='none'
          w='0px'
          opacity={0}
          transition='opacity 200ms ease-in-out'
          _focus={{
            opacity: 1,
            w: 'auto',
            transition: 'opacity 200ms ease-in-out'
          }}
        >
          <Text textStyle='header-font' whiteSpace='nowrap' fontSize='xs'>
            skip to content
          </Text>
        </Box>
      </Flex>

      <Flex>
        {/* HEADER BUTTONS */}
        <Stack display={{ base: 'none', md: 'block' }}>
          <HeaderButtons />
        </Stack>

        {/* SEARCH */}
        <Stack display={{ base: 'none', md: 'block' }} borderRight='2px' borderColor='primary'>
          <Search />
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
          aria-label={`Toggle ${isDark ? 'light' : 'dark'} mode`}
        >
          {isDark ? <SunIcon color='primary' /> : <MoonIcon color='primary' />}
        </Box>
      </Flex>

      {/* MOBILE MENU */}
      <MobileMenu />
    </Flex>
  );
};
