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

import { Code } from './UI/docs';
import { textStyles } from '../theme/foundations';

const { header1, header2, header3, header4 } = textStyles;

const MDXComponents = {
  // paragraphs
  p: ({ children }: any) => {
    return (
      <Text mb={7} lineHeight={1.5}>
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
    return (
      <Heading as='h1' textAlign='start' mb={5} {...header1}>
        {children}
      </Heading>
    );
  },
  h2: ({ children }: any) => {
    return (
      <Heading as='h2' textAlign='start' mt='16 !important' mb={4} {...header2}>
        {children}
      </Heading>
    );
  },
  h3: ({ children }: any) => {
    return (
      <Heading as='h3' mt={5} mb={2.5} {...header3}>
        {children}
      </Heading>
    );
  },
  h4: ({ children }: any) => {
    return (
      <Heading as='h4' mb={2.5} {...header4}>
        {children}
      </Heading>
    );
  },
  // tables
  table: ({ children }: any) => (
    <Flex maxW='min(100%, 100vw)' overflowX='scroll'>
      <Table
        variant='striped'
        colorScheme='greenAlpha'
        border='1px'
        borderColor='blackAlpha.50'
        my={6}
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
  ul: ({children}: any) => {
    return <UnorderedList mb={7} px={4}>{children}</UnorderedList>
  },
  ol: ({children}: any) => {
    return <OrderedList mb={7} px={4}>{children}</OrderedList>
  },
  li: ({ children }: any) => {
    return <ListItem color='primary'>{children}</ListItem>
  }
};

export default MDXComponents;
