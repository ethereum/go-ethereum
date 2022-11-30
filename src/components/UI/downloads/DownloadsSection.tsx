import { Center, Flex, IconProps, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  id: string;
  sectionTitle: string;
  sectionDescription?: React.ReactNode;
  children: React.ReactNode;
  Svg?: React.FC<IconProps>;
  ariaLabel?: string;
}

export const DownloadsSection: FC<Props> = ({
  id,
  sectionTitle,
  sectionDescription,
  children,
  Svg,
  ariaLabel
}) => {
  return (
    <Stack border='2px solid' borderColor='primary' id={id}>
      {Svg && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='primary'>
          <Svg aria-label={ariaLabel} />
        </Stack>
      )}

      <Flex
        borderBottom='2px solid'
        borderColor='primary'
        flexDirection={{ base: 'column', md: 'row' }}
      >
        <Flex p={4} sx={{ mt: '0 !important' }} flex='none'>
          <Center>
            <Text as='h2' textStyle='h2'>
              {sectionTitle}
            </Text>
          </Center>
        </Flex>

        {sectionDescription && (
          <Center
            p={4}
            borderLeft={{ base: 'none', md: '2px' }}
            borderTop={{ base: '2px', md: 'none' }}
            borderColor='primary !important'
          >
            {sectionDescription}
          </Center>
        )}
      </Flex>

      <Stack spacing={4} sx={{ mt: '0 !important' }}>
        {children}
      </Stack>
    </Stack>
  );
};
