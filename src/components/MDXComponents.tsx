import { Flex, Heading, Link, Stack, Table, Text, useColorMode } from '@chakra-ui/react';
import NextLink from 'next/link';
import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import { nightOwl, materialLight, materialDark } from 'react-syntax-highlighter/dist/cjs/styles/prism';

import bash from 'react-syntax-highlighter/dist/cjs/languages/prism/bash';
import go from 'react-syntax-highlighter/dist/cjs/languages/prism/go';
import graphql from 'react-syntax-highlighter/dist/cjs/languages/prism/graphql';
import java from 'react-syntax-highlighter/dist/cjs/languages/prism/java';
import javascript from 'react-syntax-highlighter/dist/cjs/languages/prism/javascript';
import json from 'react-syntax-highlighter/dist/cjs/languages/prism/json';
import python from 'react-syntax-highlighter/dist/cjs/languages/prism/python';
import sh from 'react-syntax-highlighter/dist/cjs/languages/prism/shell-session';
import solidity from 'react-syntax-highlighter/dist/cjs/languages/prism/solidity';
import swift from 'react-syntax-highlighter/dist/cjs/languages/prism/swift';

import { textStyles } from '../theme/foundations';
import { getProgrammingLanguageName } from '../utils';

// syntax highlighting languages supported
SyntaxHighlighter.registerLanguage('bash', bash);
SyntaxHighlighter.registerLanguage('terminal', bash);
SyntaxHighlighter.registerLanguage('go', go);
SyntaxHighlighter.registerLanguage('graphql', graphql);
SyntaxHighlighter.registerLanguage('java', java);
SyntaxHighlighter.registerLanguage('javascript', javascript);
SyntaxHighlighter.registerLanguage('json', json);
SyntaxHighlighter.registerLanguage('python', python);
SyntaxHighlighter.registerLanguage('sh', sh);
SyntaxHighlighter.registerLanguage('solidity', solidity);
SyntaxHighlighter.registerLanguage('swift', swift);


const { header1, header2, header3, header4 } = textStyles

const MDXComponents = {
  // paragraphs
  p: ({ children }: any) => {
    return (
      <Text mb={7} _last={{ mb: 0 }} size='sm' lineHeight={1.5}>
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
          color='primary'
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
      <Heading as='h2' textAlign='start' mb={4} {...header2}>
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
  // lists
  ul: ({ children }: any) => {
    return (
      <Stack as='ul' spacing={2} mb={7} _last={{ mb: 0 }}>
        {children}
      </Stack>
    );
  },
  ol: ({ children }: any) => {
    return (
      <Stack as='ol' spacing={2} mb={7} _last={{ mb: 0 }}>
        {children}
      </Stack>
    );
  },
  // tables
  table: ({ children }: any) => {
    return (
      <Flex maxW='100vw' overflowX='scroll'>
        <Table
          variant='striped'
          colorScheme='greenAlpha'
          border='1px'
          borderColor='blackAlpha.50'
          mb={10}
          size={{ base: 'sm', lg: 'md' }}
          w='auto'
        >
          {children}
        </Table>
      </Flex>
    );
  },
  // pre
  pre: ({ children }: any) => {
    return (
      <Stack mb={5}>
        <pre>{children}</pre>
      </Stack>
    );
  },
  // code
  code: ({ className, ...code }: any) => {
    const { colorMode } = useColorMode();
    const isDark = colorMode === 'dark';
    if (className?.includes("terminal")) return (
      <SyntaxHighlighter
        language='bash'
        style={nightOwl}
        customStyle={{ borderRadius: '0.5rem', padding: '1rem' }}
      >
        {code.children}
      </SyntaxHighlighter>
    )
    if (code.inline) return (
      <Text
        as={'span'}
        padding='0.125em 0.25em'
        color='primary'
        background='code-bg'
        borderRadius='0.25em'
        fontFamily='code'
        fontSize='sm'
        overflowY='scroll'
      >
        {code.children[0]}
      </Text>
    )
    if (className?.startsWith('language-')) return (
      <SyntaxHighlighter
        language='json'
        style={isDark ? materialDark : materialLight} // TODO: Update with code light/dark color themes
        customStyle={{ borderRadius: '0.5rem', padding: '1rem' }}
      >
        {code.children}
      </SyntaxHighlighter>
    )
    return (
      <Stack>
        {code.children[0]}
      </Stack>
    );
  }
};

export default MDXComponents;
