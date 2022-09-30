import { Container } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children?: React.ReactNode;
}

// TODO: if mobile, getMobileLayout, else getDesktopLayout
export const Layout: FC<Props> = ({ children }) => {
  return (
    <Container maxW={{ sm: 'container.sm' }} my={7}>
      {children}
    </Container>
  );
};
