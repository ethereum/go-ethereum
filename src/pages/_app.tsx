import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';

import 'focus-visible/dist/focus-visible';

import theme from '../theme';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ChakraProvider theme={theme}>
      <Component {...pageProps} />
    </ChakraProvider>
  );
}
