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
        </Text>
      )
    : 
      (
        <Stack>
          <ChakraCode
            overflow="scroll"
            p={6}
            background='gray.800'
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
