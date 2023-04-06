import { Box, Container, Flex, Stack } from '@chakra-ui/react';
import { FC, useContext } from 'react';
import { useRouter } from 'next/router';

import { Header } from '../UI';
import { Footer } from './Footer';
import { MobileDocsNav } from '../UI/docs';

// Context
import { NavLinksContext } from '../../context';

interface Props {
  children?: React.ReactNode;
}

export const Layout: FC<Props> = ({ children }) => {
  const router = useRouter();

  const { mobileNavLinks } = useContext(NavLinksContext);
  // console.log({ mobileNavLinks });

  return (
    <Container
      maxW={{ base: 'full', md: 'container.2xl' }}
      my={{ base: 4, md: 7 }}
      overflow='visible'
    >
      <Box
        position='sticky'
        top={{ base: 3, md: 7 }}
        backdropFilter='blur(10px)'
        opacity={0.9}
        zIndex={9}
      >
        <Header />

        {/* `MobileDocsNav` should be rendered under `/docs` pages only */}
        {router.asPath.startsWith('/docs') && (
          <Stack display={{ base: 'block', lg: 'none' }} my={6}>
            <MobileDocsNav navLinks={mobileNavLinks} />
          </Stack>
        )}
      </Box>

      {/* adding min-height & top margin to keep footer at the bottom of the page */}
      <Flex direction='column' minH='calc(100vh - 3.5rem)' height='auto' overflow='auto'>
        {children}

        <Stack mt='auto'>
          <Footer />
        </Stack>
      </Flex>
    </Container>
  );
};
