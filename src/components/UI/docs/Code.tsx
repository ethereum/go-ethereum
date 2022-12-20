// Libraries
import { Code as ChakraCode, Stack, Text, useColorMode } from '@chakra-ui/react';
import { nightOwl, prism } from 'react-syntax-highlighter/dist/cjs/styles/prism';
import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';

// Constants, utilities
import { CLASSNAME_PREFIX } from '../../../constants';
import { getProgrammingLanguageName } from '../../../utils';

// Programming lang syntax highlighters
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

interface Props {
  className: string;
  children: string[];
  inline?: boolean;
}
export const Code: React.FC<Props> = ({ className, children, inline }) => {
  const { colorMode } = useColorMode();
  const isDark = colorMode === 'dark';
  const isTerminal = className?.includes('terminal');
  const [content] = children;
  if (inline)
    return (
      <Text
        as='span'
        px={1}
        color='primary'
        bg='code-bg'
        borderRadius='0.25em'
        textStyle='inline-code-snippet'
      >
        {content}
      </Text>
    );
  if (isTerminal)
    return (
      <Stack>
        <ChakraCode overflow='auto' p={6} background='terminal-bg' color='terminal-text'>
          {content}
        </ChakraCode>
      </Stack>
    );
  if (className?.startsWith(CLASSNAME_PREFIX))
    return (
      <SyntaxHighlighter
        language={getProgrammingLanguageName(className)}
        style={isDark ? nightOwl : prism}
        customStyle={{ borderRadius: '0.5rem', padding: '1rem' }}
      >
        {content}
      </SyntaxHighlighter>
    );
  return (
    <Stack>
      <ChakraCode overflow='auto' p={6} background='terminal-bg' color='terminal-text'>
        {content}
      </ChakraCode>
    </Stack>
  );
};
