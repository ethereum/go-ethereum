// Libraries
import { Code as ChakraCode, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  code: any;
}

export const Code: FC<Props> = ({ code }) => {
  return (
    !!code.inline ?
      (
        <Text
          as='span'
          background='code-bg'
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={400}
          fontSize='md'
          lineHeight={4}
          letterSpacing='1%'
          pb={2}
          mb={-2}
        >
          {code.children[0]}
        </Text>
      )
    : 
      (
        <Stack>
          <ChakraCode
            overflow='hidden'
            p={6}
            background='code-bg-contrast'
            color='green.50'
            fontFamily='"JetBrains Mono", monospace'
            fontWeight={400}
            fontSize='md'
            lineHeight='21.12px'
            letterSpacing='1%'
          >
            {code.children[0]}
          </ChakraCode>
        </Stack>
      )
  );
};
