import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';
import { MDXProvider } from '@mdx-js/react';

import { Layout } from '../components/layouts';

import MDComponents from '../components/UI/docs';

import 'focus-visible/dist/focus-visible';
import theme from '../theme';

// Algolia search css styling
import '../theme/search.css';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ChakraProvider theme={theme}>
      <MDXProvider components={MDComponents}>
        <Layout>
          <Component {...pageProps} />
        </Layout>
      </MDXProvider>
    </ChakraProvider>
  );
}
