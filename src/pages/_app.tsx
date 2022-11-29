import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';
import Head from 'next/head';

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
          <Head>
            <meta name='viewport' content='width=device-width, initial-scale=1' />
            <link rel='icon' type='image/x-icon' href='/favicon.ico' />
          </Head>

          <Component {...pageProps} />
        </Layout>
      </MDXProvider>
    </ChakraProvider>
  );
}
