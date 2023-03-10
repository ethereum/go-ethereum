import { Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  os: string;
}

export const BuildsDeprecationNote: FC<Props> = ({ os }) => {
  return (
    <Stack p={4}>
      <Text textAlign='center' bg='code-bg' p={3}>
        <strong>Geth no longer releases builds for {os}.</strong> {os} builds on this page are
        archival and are not consistent with current Geth.
      </Text>
    </Stack>
  );
};
