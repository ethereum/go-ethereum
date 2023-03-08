import { Center, Flex, Link, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import {
  DOCS_PAGE,
  DOWNLOADS_PAGE,
  GETH_DISCORD_URL,
  GETH_REPO_URL,
  GETH_TWITTER_URL
} from '../../constants';

import { DiscordIcon, GitHubIcon, TwitterIcon } from '../UI/icons';

const hoverStyles = {
  textDecoration: 'none',
  bg: 'primary',
  color: 'bg !important'
};

export const Footer: FC = () => {
  return (
    <Flex mt={{ base: 4, lg: 8 }} direction={{ base: 'column', lg: 'row' }}>
      <Flex
        direction={{ base: 'column', md: 'row' }}
        justifyContent={{ md: 'space-between' }}
        border='2px solid'
        borderColor='primary'
      >
        <Flex
          sx={{ mt: '-2px !important' }}
          borderBottom={{
            base: '2px solid',
            md: 'none'
          }}
          borderColor='primary'
        >
          <Center
            flex={1}
            color='primary'
            _hover={hoverStyles}
            borderRight='2px solid'
            borderColor='primary'
          >
            <NextLink href={DOWNLOADS_PAGE} passHref legacyBehavior>
              <Link _hover={{ textDecoration: 'none' }} p={4}>
                <Text textStyle='footer-link-label'>DOWNLOADS</Text>
              </Link>
            </NextLink>
          </Center>

          <Center
            flex={1}
            color='primary'
            _hover={hoverStyles}
            borderRight={{
              base: 'none',
              md: '2px solid'
            }}
            borderColor='primary'
          >
            <NextLink href={DOCS_PAGE} passHref legacyBehavior>
              <Link _hover={{ textDecoration: 'none' }} p={4}>
                <Text textStyle='footer-link-label'>DOCUMENTATION</Text>
              </Link>
            </NextLink>
          </Center>
        </Flex>

        <Flex sx={{ mt: '0 !important' }}>
          <Center
            flex={1}
            data-group
            borderLeft={{
              base: 'none',
              md: '2px solid',
              lg: 'none'
            }}
            borderColor='primary !important'
            _hover={hoverStyles}
          >
            <Link href={GETH_TWITTER_URL} isExternal p={4}>
              <TwitterIcon
                w={8}
                height='22px'
                _groupHover={{ color: 'bg' }}
                color='primary'
                aria-label='Twitter'
              />
            </Link>
          </Center>

          <Center
            data-group
            flex={1}
            _hover={hoverStyles}
            borderWidth='2px'
            borderStyle='none solid'
            borderColor='primary'
          >
            <Link href={GETH_DISCORD_URL} isExternal p={4}>
              <DiscordIcon
                w={8}
                height='22px'
                _groupHover={{ color: 'bg' }}
                color='primary'
                aria-label='Discord'
              />
            </Link>
          </Center>

          <Center data-group flex={1} _hover={hoverStyles}>
            <Link href={GETH_REPO_URL} isExternal p={4}>
              <GitHubIcon
                w={7}
                height='22px'
                _groupHover={{ color: 'bg' }}
                color='primary'
                aria-label='GitHub'
              />
            </Link>
          </Center>
        </Flex>
      </Flex>

      <Center
        p={4}
        borderWidth='2px'
        borderStyle={{
          base: 'none solid solid solid',
          lg: 'solid solid solid none'
        }}
        borderColor='primary'
        flex={1}
      >
        <Text textStyle='footer-text'>
          {`© 2013–${new Date().getFullYear()}. The go-ethereum Authors`} |{' '}
          <Link isExternal variant='light' href={process.env.NEXT_PUBLIC_MATOMO_OPT_OUT_URL}>
            Do-not-Track
          </Link>
        </Text>
      </Center>
    </Flex>
  );
};
