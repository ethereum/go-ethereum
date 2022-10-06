import { Heading, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children?: React.ReactNode;
  imgSrc?: string;
  imgAltText?: string;
  sectionTitle: string
}

export const DownloadsSection: FC<Props> = ({
  children,
  imgSrc,
  imgAltText,
  sectionTitle,
}) => {
  return (
    <Stack border='2px solid #11866F'>
      {!!imgSrc && (
        <Stack alignItems='center' p={4} borderBottom='2px solid #11866f'>
          {/* TODO: use NextImage */}
          <Image src={imgSrc} alt={imgAltText} />
        </Stack>
      )}

      <Stack p={4} borderBottom='2px solid #11866f' sx={{ mt: '0 !important' }}>
        <Heading
          // TODO: move text style to theme
          as='h2'
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={400}
          fontSize='1.5rem'
          lineHeight='auto'
          letterSpacing='4%'
          // TODO: move to theme colors
          color='#1d242c'
        >
          {sectionTitle}
        </Heading>
      </Stack>

      <Stack spacing={4}>
        {children}
      </Stack>
    </Stack>
  )
}