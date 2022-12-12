import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';

import { Layout } from '../components/layouts';

import 'focus-visible/dist/focus-visible';
import theme from '../theme';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ChakraProvider theme={theme}>
      <Layout>
        <Component {...pageProps} />
      </Layout>
    </ChakraProvider>
  );
}
