import { Container } from '@chakra-ui/react';
import { FC } from 'react';

import { Header } from '../UI';

interface Props {
  children?: React.ReactNode;
}

export const Layout: FC<Props> = ({ children }) => {
  return (
    <Container maxW={{ base: 'container.sm', md: 'container.2xl' }} my={7}>
      <Header />

      {children}
    </Container>
  );
};
