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
          textStyle='inline-code-snippet'
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
            overflow='auto'
            p={6}
            background='code-bg-contrast'
            textStyle='code-block'
            color='code-text'
          >
            {code.children[0]}
          </ChakraCode>
        </Stack>
      )
  );
};
