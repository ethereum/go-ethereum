import { Center, Flex, Image, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children: React.ReactNode;
  id: string;
  imgSrc?: string;
  imgAltText?: string;
  sectionDescription?: React.ReactNode;
  sectionTitle: string;
}

export const DownloadsSection: FC<Props> = ({ children, imgSrc, imgAltText, sectionDescription, sectionTitle, id }) => {
  return (
    <Stack border='2px solid' borderColor='brand.light.primary' id={id}>
      {!!imgSrc && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
          {/* TODO: use NextImage */}
          <Image src={imgSrc} alt={imgAltText} />
        </Stack>
      )}

      <Flex
        borderBottom='2px solid'
        borderColor='brand.light.primary'
        flexDirection={{base: 'column', md: 'row'}}
      >
        <Flex
          p={4}
          sx={{ mt: '0 !important' }}
          flex='none'
        >
          <Center>
            <Text as='h2' textStyle='h2'>
              {sectionTitle}
            </Text>
          </Center>
        </Flex>
        
        {
          sectionDescription && (
            <Stack
              p={4}
              borderLeft={{ base: 'none', md: '2px solid #11866f'}}
              borderTop={{ base: '2px solid #11866f', md: 'none'}}
            >
              <Center>
                {sectionDescription}
              </Center>
            </Stack>
          )
        }
      </Flex>

      <Stack spacing={4} sx={{ mt: '0 !important' }} >{children}</Stack>
    </Stack>
  );
};
