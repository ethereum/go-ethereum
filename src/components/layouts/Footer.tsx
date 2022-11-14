import { Flex, Link, Stack, Text } from '@chakra-ui/react';
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

export const Footer: FC = () => {
  return (
    <Flex mt={4} direction={{ base: 'column', lg: 'row' }}>
      <Flex
        direction={{ base: 'column', md: 'row' }}
        justifyContent={{ md: 'space-between' }}
        border='2px solid'
        borderColor='brand.light.primary'
      >
        <Flex
          borderBottom={{
            base: '2px solid',
            md: 'none'
          }}
          borderColor='brand.light.primary'
        >
          <NextLink href={DOWNLOADS_PAGE} passHref>
            <Link
              flex={1}
              color='brand.light.primary'
              _hover={{
                textDecoration: 'none',
                bg: 'brand.light.primary',
                color: 'yellow.50 !important'
              }}
              height='full'
              borderRight='2px solid'
              borderColor='brand.light.primary'
            >
              <Text textStyle='home-section-link-label'>DOWNLOADS</Text>
            </Link>
          </NextLink>

          <NextLink href={DOCS_PAGE} passHref>
            <Link
              flex={1}
              color='brand.light.primary'
              _hover={{
                textDecoration: 'none',
                bg: 'brand.light.primary',
                color: 'yellow.50 !important'
              }}
              height='full'
              borderRight={{
                base: 'none',
                md: '2px solid'
              }}
              borderColor='brand.light.primary'
            >
              <Text textStyle='home-section-link-label'>DOCUMENTATION</Text>
            </Link>
          </NextLink>
        </Flex>

        <Flex>
          <NextLink href={GETH_TWITTER_URL} passHref>
            <Link
              isExternal
              p={4}
              display='flex'
              flex={1}
              data-group
              borderLeft={{
                base: 'none',
                md: '2px solid',
                lg: 'none'
              }}
              borderColor='brand.light.primary !important'
              _hover={{
                bg: 'brand.light.primary'
              }}
              justifyContent='center'
            >
              <TwitterIcon
                w={6}
                height={6}
                margin='auto'
                _groupHover={{
                  svg: {
                    path: { fill: 'yellow.50 !important' }
                  }
                }}
              />
            </Link>
          </NextLink>

          <NextLink href={GETH_DISCORD_URL} passHref>
            <Link
              isExternal
              p={4}
              data-group
              display='flex'
              flex={1}
              _hover={{
                bg: 'brand.light.primary'
              }}
              justifyContent='center'
              borderWidth='2px'
              borderStyle='none solid'
              borderColor='brand.light.primary'
            >
              <DiscordIcon
                w={6}
                height={6}
                _groupHover={{
                  svg: {
                    path: { fill: 'yellow.50 !important' }
                  }
                }}
              />
            </Link>
          </NextLink>

          <NextLink href={GETH_REPO_URL} passHref>
            <Link
              isExternal
              p={4}
              data-group
              flex={1}
              display='flex'
              _hover={{
                bg: 'brand.light.primary'
              }}
              justifyContent='center'
            >
              <GitHubIcon
                w={6}
                height={6}
                _groupHover={{
                  svg: {
                    path: { fill: 'yellow.50 !important' }
                  }
                }}
              />
            </Link>
          </NextLink>
        </Flex>
      </Flex>

      <Stack
        p={4}
        textAlign='center'
        justifyContent='center'
        borderWidth='2px'
        borderStyle={{
          base: 'none solid solid solid',
          lg: 'solid solid solid none'
        }}
        borderColor='brand.light.primary'
        flex={1}
      >
        <Text textStyle='footer-text'>© 2013–2022. The go-ethereum Authors.</Text>
      </Stack>
    </Flex>
  );
};
