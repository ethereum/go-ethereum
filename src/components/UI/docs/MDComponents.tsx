import {
  Flex,
  Heading,
  Link,
  ListItem,
  OrderedList,
  Stack,
  Table,
  Text,
  UnorderedList
} from '@chakra-ui/react';
import NextLink from 'next/link';

import { Code, Note } from '.';
import { textStyles } from '../../../theme/foundations';
import { parseHeadingId } from '../../../utils/parseHeadingId';

const { header1, header2, header3, header4 } = textStyles;

const MDComponents = {
  // paragraphs
  p: ({ children }: any) => {
    return (
      <Text mb='7 !important' lineHeight={1.5}>
        {children}
      </Text>
    );
  },
  // links
  a: ({ children, href }: any) => {
    return (
      <NextLink href={href} passHref>
        <Link
          isExternal={href.startsWith('http') && !href.includes('geth.ethereum.org')}
          variant='light'
        >
          {children}
        </Link>
      </NextLink>
    );
  },
  // headings
  h1: ({ children }: any) => {
    const heading = parseHeadingId(children);

    return heading ? (
      <Heading as='h1' textAlign='start' mb='5 !important' {...header1} id={heading.headingId}>
        {heading.children}
      </Heading>
    ) : (
      <Heading as='h1' textAlign='start' mb='5 !important' {...header1}>
        {children}
      </Heading>
    );
  },
  h2: ({ children }: any) => {
    const heading = parseHeadingId(children);

    return heading ? (
      <Heading
        as='h2'
        textAlign='start'
        mt='16 !important'
        mb='4 !important'
        {...header2}
        id={heading.headingId}
      >
        {heading.children}
      </Heading>
    ) : (
      <Heading as='h2' textAlign='start' mt='16 !important' mb='4 !important' {...header2}>
        {children}
      </Heading>
    );
  },
  h3: ({ children }: any) => {
    const heading = parseHeadingId(children);

    return heading ? (
      <Heading as='h3' mt='5 !important' mb='2.5 !important' {...header3} id={heading.headingId}>
        {heading.children}
      </Heading>
    ) : (
      <Heading as='h3' mt='5 !important' mb='2.5 !important' {...header3}>
        {children}
      </Heading>
    );
  },
  h4: ({ children }: any) => {
    const heading = parseHeadingId(children);

    return heading ? (
      <Heading as='h4' mb='2.5 !important' {...header4} id={heading.headingId}>
        {heading.children}
      </Heading>
    ) : (
      <Heading as='h4' mb='2.5 !important' {...header4}>
        {children}
      </Heading>
    );
  },
  // tables
  table: ({ children }: any) => (
    <Flex maxW='min(100%, 100vw)' overflowX='auto'>
      <Table
        variant='striped'
        colorScheme='greenAlpha'
        border='1px'
        borderColor='blackAlpha.50'
        my='6 !important'
        size={{ base: 'sm', lg: 'md' }}
        w='auto'
      >
        {children}
      </Table>
    </Flex>
  ),
  // pre
  pre: ({ children }: any) => (
    <Stack mb={5}>
      <pre>{children}</pre>
    </Stack>
  ),
  // code
  code: ({ children, ...props }: any) => <Code {...props}>{children}</Code>,
  // list
  ul: ({ children }: any) => {
    return (
      <Stack>
        <UnorderedList mb={7} px={4}>
          {children}
        </UnorderedList>
      </Stack>
    );
  },
  ol: ({ children }: any) => {
    return (
      <Stack>
        <OrderedList mb={7} px={4}>
          {children}
        </OrderedList>
      </Stack>
    );
  },
  li: ({ children }: any) => {
    return <ListItem color='primary'>{children}</ListItem>;
  },
  note: ({ children }: any) => {
    return <Note>{children}</Note>;
  }
};

export default MDComponents;
