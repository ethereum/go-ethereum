import { Box, Grid, GridItem, Link, Stack, Text } from '@chakra-ui/react';
import type { NextPage } from 'next';

import {
  HomeHero,
  HomeSection,
  QuickLinks,
  WhatIsEthereum,
  WhyRunANode
} from '../components/UI/homepage';
import { PageMetadata } from '../components/UI';
import { GopherHomeFront, GopherHomeLinks } from '../components/UI/svgs';

import {
  CONTRIBUTING_PAGE,
  DOCS_PAGE,
  ETHEREUM_FOUNDATION_URL,
  ETHEREUM_ORG_URL,
  GETH_REPO_URL,
  GO_URL,
  METADATA
} from '../constants';

const HomePage: NextPage = ({}) => {
  return (
    <>
      <PageMetadata title={METADATA.HOME_TITLE} description={METADATA.HOME_DESCRIPTION} />

      <main id='main-content'>
        <Stack spacing={{ base: 4, lg: 8 }}>
          <HomeHero />

          <Grid
            templateColumns={{ base: 'repeat(1, 1fr)', lg: 'repeat(2, 1fr)' }}
            gap={{ base: 4, lg: 8 }}
          >
            <GridItem rowSpan={2}>
              {/* SECTION: What is Geth */}
              <HomeSection
                sectionTitle='What is Geth?'
                linkLabel='Get started with Geth'
                buttonHref={`${DOCS_PAGE}/getting-started`}
                Svg={GopherHomeFront}
                ariaLabel='Gopher greeting'
              >
                <Text textStyle='quick-link-text'>
                  Geth (go-ethereum) is a{' '}
                  <Link href={GO_URL} isExternal variant='light' aria-label='Go lang'>
                    Go
                  </Link>{' '}
                  implementation of{' '}
                  <Link href={ETHEREUM_ORG_URL} isExternal variant='light'>
                    Ethereum
                  </Link>{' '}
                  - a gateway into the decentralized web.
                </Text>

                <Text textStyle='quick-link-text'>
                  Geth has been a core part of Ethereum since the very beginning. Geth was one of
                  the original Ethereum implementations making it the most battle-hardened and
                  tested client.
                </Text>

                <Text textStyle='quick-link-text'>
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

                <Text textStyle='quick-link-text'>
                  Running Geth alongside a consensus client turns a computer into an Ethereum node.
                </Text>
              </HomeSection>
            </GridItem>

            <GridItem>
              {/* SECTION: What is Ethereum (has different styles than the other sections so it uses a different component) */}
              <WhatIsEthereum>
                <Text textStyle='quick-link-text'>
                  Ethereum is a technology for building apps and organizations, holding assets,
                  transacting and communicating without being controlled by a central authority. It
                  is the base of a new, decentralized internet.
                </Text>
              </WhatIsEthereum>
            </GridItem>

            <GridItem>
              {/* SECTION: Why run a node (has different styles than the other sections so it uses a different component) */}
              <WhyRunANode>
                <Text textStyle='quick-link-text'>
                  Running your own node enables you to use Ethereum in a truly private,
                  self-sufficient and trustless manner. You don&apos;t need to trust information you
                  receive because you can verify the data yourself using your Geth instance.
                </Text>

                <Text textStyle='quick-link-text' fontWeight={700}>
                  &ldquo;Don&apos;t trust, verify&rdquo;
                </Text>
              </WhyRunANode>
            </GridItem>
          </Grid>

          <Grid
            templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)' }}
            gap={{ base: 4, lg: 8 }}
          >
            <GridItem>
              {/* SECTION: Contribute to Geth */}
              <HomeSection
                sectionTitle='Contribute to Geth'
                linkLabel='Read our contribution guidelines'
                buttonHref={CONTRIBUTING_PAGE}
              >
                <Text textStyle='quick-link-text'>
                  We welcome contributions from anyone on the internet, and are grateful for even
                  the smallest of fixes! If you&apos;d like to contribute to the Geth source code,
                  please fork the{' '}
                  <Link href={GETH_REPO_URL} isExternal variant='light'>
                    GitHub repository
                  </Link>
                  , fix, commit and send a pull request for the maintainers to review and merge into
                  the main code base.
                </Text>
              </HomeSection>
            </GridItem>

            <GridItem>
              {/* SECTION: About the Team */}
              <HomeSection
                sectionTitle='About the Team'
                linkLabel='Read more about the Ethereum Foundation'
                buttonHref={ETHEREUM_FOUNDATION_URL}
              >
                <Text textStyle='quick-link-text'>
                  The Geth team comprises 10 developers distributed across the world. The Geth team
                  is funded exclusively by The Ethereum Foundation.
                </Text>
              </HomeSection>
            </GridItem>
          </Grid>

          <Grid templateColumns={{ base: '1fr', md: '300px 1fr' }} gap={{ base: 4, lg: 8 }}>
            <GridItem w='auto'>
              <Box h='100%'>
                {/* TODO: replace with animated/video version */}
                <Stack
                  justifyContent='center'
                  alignItems='center'
                  p={4}
                  border='2px solid'
                  borderColor='primary'
                  h='100%'
                >
                  <GopherHomeLinks />
                </Stack>
              </Box>
            </GridItem>

            <GridItem>
              <QuickLinks />
            </GridItem>
          </Grid>
        </Stack>
      </main>
    </>
  );
};

export default HomePage;
