import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';
import { MDXProvider } from '@mdx-js/react';

import { Layout } from '../components/layouts';

import MDXComponents from '../components/';

import 'focus-visible/dist/focus-visible';
import theme from '../theme';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ChakraProvider theme={theme}>
      <MDXProvider components={MDXComponents}>
        <Layout>
          <Component {...pageProps} />
        </Layout>
      </MDXProvider>
    </ChakraProvider>
  );
}
