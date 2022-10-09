import { Heading, Link, Stack, Text } from '@chakra-ui/react';
import NextLink from 'next/link';
import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import { nightOwl } from 'react-syntax-highlighter/dist/cjs/styles/prism';

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

// syntax highlighting languages supported
SyntaxHighlighter.registerLanguage('bash', bash);
SyntaxHighlighter.registerLanguage('go', go);
SyntaxHighlighter.registerLanguage('graphql', graphql);
SyntaxHighlighter.registerLanguage('java', java);
SyntaxHighlighter.registerLanguage('javascript', javascript);
SyntaxHighlighter.registerLanguage('json', json);
SyntaxHighlighter.registerLanguage('python', python);
SyntaxHighlighter.registerLanguage('sh', sh);
SyntaxHighlighter.registerLanguage('solidity', solidity);
SyntaxHighlighter.registerLanguage('swift', swift);

import { getProgrammingLanguageName } from '../utils';

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
          // TODO: move color to theme
          color='#18bc9c'
        >
          {children}
        </Link>
      </NextLink>
    );
  },
  // headings
  h1: ({ children }: any) => {
    return (
      <Heading as='h1' textAlign='start' fontSize='4xl' mb={5}>
        {children}
      </Heading>
    );
  },
  h2: ({ children }: any) => {
    return (
      <Heading as='h2' textAlign='start' fontSize='3xl' mb={4}>
        {children}
      </Heading>
    );
  },
  h3: ({ children }: any) => {
    return (
      <Heading as='h3' fontSize='2xl' mt={5} mb={2.5}>
        {children}
      </Heading>
    );
  },
  h4: ({ children }: any) => {
    return (
      <Heading as='h4' fontSize='lg' mb={2.5}>
        {children}
      </Heading>
    );
  },
  // pre
  pre: ({ children }: any) => {
    return (
      <Stack mb={5}>
        <pre>{children}</pre>
      </Stack>
    );
  }
  // code
  // code: (code: any) => {
  //   const language = getProgrammingLanguageName(code);

  //   return !!code.inline ? (
  //     <Text
  //       as={'span'}
  //       padding='0.125em 0.25em'
  //       color='red.300'
  //       background='#1c1e2d'
  //       borderRadius='0.25em'
  //       fontFamily='code'
  //       fontSize='sm'
  //       overflowY='scroll'
  //     >
  //       {code.children[0]}
  //     </Text>
  //   ) : (
  //     <Stack style={nightOwl}>
  //       {code.children[0]}
  //     </Stack>
  //   );
  // }
};

export default MDXComponents;
