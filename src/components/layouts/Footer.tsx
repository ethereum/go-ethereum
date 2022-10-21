import { Box, Flex, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import {
  DOCS_PAGE,
  DOWNLOADS_PAGE,
  GETH_DISCORD_URL,
  GETH_REPO_URL,
  GETH_TWITTER_URL
} from '../../constants'

import {
  DiscordIcon,
  GitHubIcon,
  TwitterIcon
} from '../UI/icons';

export const Footer: FC = () => {
  return (
    <Box mt={4} border='2px solid' borderColor='brand.light.primary'>
      <Flex
        direction={{ base: 'column'}}
      >
        <Flex sx={{ mt: '-2px !important' }}>
          <Stack
            flex={1}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
            color='brand.light.primary'
              _hover={{
                textDecoration: 'none',
                bg: 'brand.light.primary',
                color: 'yellow.50 !important'
              }}
          >
            <NextLink href={DOWNLOADS_PAGE} passHref>
              <Link _hover={{ textDecoration: 'none' }}>
                <Text textStyle='home-section-link-label'>DOWNLOADS</Text>
              </Link>
            </NextLink>
          </Stack>

          <Stack
            flex={1}
            borderBottom='2px solid'
            borderColor='brand.light.primary'
            color='brand.light.primary'
              _hover={{
                textDecoration: 'none',
                bg: 'brand.light.primary',
                color: 'yellow.50 !important'
              }}
          >
            <NextLink href={DOCS_PAGE} passHref>
              <Link _hover={{ textDecoration: 'none' }}>
                <Text textStyle='home-section-link-label'>DOCUMENTATION</Text>
              </Link>
            </NextLink>
          </Stack>
        </Flex>

        <Flex sx={{ mt: '0 !important' }}>
          <Stack
            flex={1}
            data-group
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
            _hover={{
              bg: 'brand.light.primary',
            }}
            alignItems='center'
          >
            <NextLink href={GETH_TWITTER_URL} passHref>
              <Link isExternal p={4}>
                <TwitterIcon
                  w={8}
                  height={8} 
                  _groupHover={{
                    svg: {
                      path:{fill: 'yellow.50 !important'}
                    }
                  }}
                />
              </Link>
            </NextLink>
          </Stack>

          <Stack
            data-group
            flex={1}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
            _hover={{
              bg: 'brand.light.primary',
            }}
            alignItems='center'
          >
            <NextLink href={GETH_DISCORD_URL} passHref>
              <Link isExternal p={4}>
                <DiscordIcon
                  w={8}
                  height={8} 
                  _groupHover={{
                    svg: {
                      path:{fill: 'yellow.50 !important'}
                    }
                  }}
                />
              </Link>
            </NextLink>
          </Stack>

          <Stack
            data-group
            flex={1}
            borderBottom='2px solid'
            borderColor='brand.light.primary'
            _hover={{
              bg: 'brand.light.primary',
            }}
            alignItems='center'
          >
            <NextLink href={GETH_REPO_URL} passHref>
              <Link isExternal p={4}>
                <GitHubIcon
                  w={7}
                  height={7} 
                  _groupHover={{
                    svg: {
                      path:{fill: 'yellow.50 !important'}
                    }
                  }}
                />
              </Link>
            </NextLink>
          </Stack>
        </Flex>
      </Flex>

      <Stack p={4} sx={{ mt: '0 !important' }} textAlign='center'>
        <Text textStyle='footer-text'>© 2013–2022. The go-ethereum Authors.</Text>
      </Stack>
    </Box>
  )
}