import { FC } from 'react';
import { Stack, Text } from '@chakra-ui/react';

interface Props {
  children: string[];
}

export const Note: FC<Props> = ({ children }) => {
  return (
    <Stack w='100%' bg='button-bg' border='2px' borderColor='primary' p={4}>
      <Text as='h4' textStyle='header4'>
        Note
      </Text>
      <Text textStyle='note-text'>{children}</Text>
    </Stack>
  );
};
