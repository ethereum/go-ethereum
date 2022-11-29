import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';

import { Layout } from '../components/layouts';

import 'focus-visible/dist/focus-visible';

import theme from '../theme';

import { MDXProvider } from '@mdx-js/react';
import MDXComponents from '../components/';

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
