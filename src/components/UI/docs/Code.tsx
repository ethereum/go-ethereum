// Libraries
import { Code as ChakraCode } from '@chakra-ui/react';
import { FC } from 'react';
 
// Utils
import { getProgrammingLanguageName } from '../../../utils';


interface Props {
  code: any;
}

export const Code: FC<Props> = ({ code }) => {
  const language = getProgrammingLanguageName(code);

  return (
    !!code.inline ?
      (
        <ChakraCode
          background='gray.200'
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={400}
          fontSize='md'
          lineHeight={4}
          letterSpacing='1%'
          pb={2}
          mb={-2}
        >
          {code.children[0]}
        </ChakraCode>
      )
    : 
      (
        <p>test</p>
      )
  );
};
