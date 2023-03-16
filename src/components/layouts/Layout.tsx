// Libraries
import { Container, Flex, Stack } from '@chakra-ui/react';
import { FC } from 'react';

// Components
import { Header } from '../UI';
import { Footer } from './Footer';

interface Props {
  children?: React.ReactNode;
}

export const Layout: FC<Props> = ({ children }) => {
  return (
    <Container maxW={{ base: 'full', md: 'container.2xl' }} my={{ base: 4, md: 7 }}>
      {/* adding min-height & top margin to keep footer at the bottom of the page */}
      <Flex direction='column' minH='calc(100vh - 3.5rem)'>
        <Header />

        {children}

        <Stack mt='auto'>
          <Footer />
        </Stack>
      </Flex>
    </Container>
  );
};
