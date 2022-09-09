import { Container } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children?: React.ReactNode;
}

export const Layout: FC<Props> = ({ children }) => {
  return (
    <Container maxW='container.lg' my={7}>
      {children}
    </Container>
  );
};
