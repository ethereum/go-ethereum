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

const { h1, h2, h3, h4 } = textStyles;

const MDComponents = {
  // paragraphs
  p: ({ children }: any) => {
    return (
      <Text mb='7 !important' lineHeight={1.6}>
        {children}
      </Text>
    );
  },
  // links
  a: ({ children, href }: any) => {
    const isExternal = href.startsWith('http') && !href.includes('geth.ethereum.org');

    return isExternal ? (
      <Link href={href} isExternal variant='light'>
        {children}
      </Link>
    ) : (
      <NextLink href={href} passHref legacyBehavior>
        <Link variant='light'>{children}</Link>
      </NextLink>
    );
  },
  // headings
  h1: ({ children }: any) => {
    const { children: parsedChildren, headingId } = parseHeadingId(children);

    return (
      <Heading as='h1' textAlign='start' mb='5 !important' {...h1} id={headingId}>
        {parsedChildren}
      </Heading>
    );
  },
  h2: ({ children }: any) => {
    const { children: parsedChildren, headingId } = parseHeadingId(children);

    return (
      <Heading
        as='h2'
        textAlign='start'
        mt={{ base: '12 !important', md: '16 !important' }}
        mb='4 !important'
        {...h2}
        id={headingId}
      >
        {parsedChildren}
      </Heading>
    );
  },
  h3: ({ children }: any) => {
    const { children: parsedChildren, headingId } = parseHeadingId(children);
    return (
      <Heading as='h3' mt='5 !important' mb='2.5 !important' {...h3} id={headingId}>
        {parsedChildren}
      </Heading>
    );
  },
  h4: ({ children }: any) => {
    const { children: parsedChildren, headingId } = parseHeadingId(children);

    return (
      <Heading as='h4' mb='2.5 !important' {...h4} id={headingId}>
        {parsedChildren}
      </Heading>
    );
  },
  // tables
  table: ({ children }: any) => (
    <Flex overflowX='auto'>
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
    <Stack mb={5} whiteSpace='pre'>
      {children}
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
    return <ListItem>{children}</ListItem>;
  },
  note: ({ children }: any) => {
    return <Note>{children}</Note>;
  }
};

export default MDComponents;
