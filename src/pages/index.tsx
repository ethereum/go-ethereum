import { Link, Stack, Text } from '@chakra-ui/react';
import type { NextPage } from 'next';

import { Gopher, HomeHero, HomeSection, QuickLinks } from '../components/UI/homepage';

import {
  CONTRIBUTING_PAGE,
  DOCS_PAGE,
  ETHEREUM_FOUNDATION_URL,
  ETHEREUM_ORG_RUN_A_NODE_URL,
  ETHEREUM_ORG_URL,
  GETH_REPO_URL
} from '../constants';

const HomePage: NextPage = ({}) => {
  return (
    <>
      {/* TODO: add PageMetadata */}

      <main>
        <Stack spacing={4}>
          <HomeHero />

          {/* SECTION: What is Geth */}
          <HomeSection
            imgSrc='/images/pages/gopher-home-front.svg'
            imgAltText='Gopher greeting'
            sectionTitle='What is Geth'
            linkLabel='Get started with Geth'
            buttonHref={`${DOCS_PAGE}/getting-started`}
          >
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Geth (go-ethereum) is a{' '}
              <Link
                href='https://go.dev/'
                isExternal
                textDecoration='underline'
                color='brand.light.primary'
                _hover={{ color: 'brand.light.body', textDecorationColor: 'brand.light.body' }}
                _focus={{
                  color: 'brand.light.primary',
                  boxShadow: '0 0 0 1px #11866f !important',
                  textDecoration: 'none'
                }}
                _pressed={{
                  color: 'brand.light.secondary',
                  textDecorationColor: 'brand.light.secondary'
                }}
              >
                Go
              </Link>{' '}
              implementation of{' '}
              <Link
                href={ETHEREUM_ORG_URL}
                isExternal
                textDecoration='underline'
                color='brand.light.primary'
                _hover={{ color: 'brand.light.body', textDecorationColor: 'brand.light.body' }}
                _focus={{
                  color: 'brand.light.primary',
                  boxShadow: '0 0 0 1px #11866f !important',
                  textDecoration: 'none'
                }}
                _pressed={{
                  color: 'brand.light.secondary',
                  textDecorationColor: 'brand.light.secondary'
                }}
              >
                Ethereum
              </Link>{' '}
              - a gateway into the decentralized web.
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Geth has been a core part of Etheruem since the very beginning. Geth was one of the
              original Ethereum implementations making it the most battle-hardened and tested
              client.
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Geth is an Ethereum{' '}
              <Text as='span' fontStyle='italic'>
                execution client
              </Text>{' '}
              meaning it handles transactions, deployment and execution of smart contracts and
              contains an embedded computer known as the{' '}
              <Text as='span' fontStyle='italic'>
                Ethereum Virtual Machine
              </Text>
              .
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Running Geth alongside a consensus client turns a computer into an Ethereum node.
            </Text>
          </HomeSection>

          {/* SECTION: What is Ethereum */}
          <HomeSection
            imgSrc='/images/pages/glyph-home-light.svg'
            imgAltText='Ethereum glyph'
            sectionTitle='What is Ethereum'
            linkLabel='Learn more on Ethereum.org'
            buttonHref={ETHEREUM_ORG_URL}
          >
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Ethereum is a technology for building apps and organizations, holding assets,
              transacting and communicating without being controlled by a central authority. It is
              the base of a new, decentralized internet.
            </Text>
          </HomeSection>

          {/* SECTION: Why run a Node */}
          <HomeSection
            imgSrc='/images/pages/gopher-home-nodes.svg'
            imgAltText='Gopher staring at nodes'
            sectionTitle='Why run a node?'
            linkLabel='Read more about running a node'
            buttonHref={ETHEREUM_ORG_RUN_A_NODE_URL}
          >
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Running your own node enables you to use Ethereum in a truly private, self-sufficient
              and trustless manner. You don&apos;t need to trust information you receive because you
              can verify the data yourself using your Geth instance.
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px' fontWeight={700}>
              &ldquo;Don&apos;t trust, verify&rdquo;
            </Text>
          </HomeSection>

          {/* SECTION:Contribute to Geth */}
          <HomeSection
            sectionTitle='Contribute to Geth'
            linkLabel='Read our contribution guidelines'
            buttonHref={CONTRIBUTING_PAGE}
          >
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              We welcome contributions from anyone on the internet, and are grateful for even the
              smallest of fixes! If you&apos;d like to contribute to the Geth source code, please
              fork the{' '}
              <Link
                href={GETH_REPO_URL}
                isExternal
                textDecoration='underline'
                color='brand.light.primary'
                _hover={{ color: 'brand.light.body', textDecorationColor: 'brand.light.body' }}
                _focus={{
                  color: 'brand.light.primary',
                  boxShadow: '0 0 0 1px #11866f !important',
                  textDecoration: 'none'
                }}
                _pressed={{
                  color: 'brand.light.secondary',
                  textDecorationColor: 'brand.light.secondary'
                }}
              >
                Github repository
              </Link>
              , fix, commit and send a pull request for the maintainers to review and merge into the
              main code base.
            </Text>
          </HomeSection>

          {/* SECTION: About the Team */}
          <HomeSection
            sectionTitle='About the Team'
            linkLabel='Read more about the Ethereum Foundation'
            buttonHref={ETHEREUM_FOUNDATION_URL}
          >
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              The Geth team comprises 10 developers distributed across the world. The Geth team is
              funded directly by The Ethereum Foundation.
            </Text>
          </HomeSection>

          {/* TODO: replace with animated/video version */}
          <Gopher />

          <QuickLinks />
        </Stack>
      </main>
    </>
  );
};

export default HomePage;
