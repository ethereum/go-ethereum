import { ChakraProvider } from '@chakra-ui/react';
import { AppProps } from 'next/app';
import { useEffect } from 'react';
import { init } from '@socialgouv/matomo-next';

import { Layout } from '../components/layouts';

import 'focus-visible/dist/focus-visible';
import theme from '../theme';

// Algolia search css styling
import '../theme/search.css';

export default function App({ Component, pageProps }: AppProps) {
  useEffect(() => {
    init({
      url: process.env.NEXT_PUBLIC_MATOMO_URL!,
      siteId: process.env.NEXT_PUBLIC_MATOMO_SITE_ID!
    });
  }, []);

  return (
    <ChakraProvider theme={theme}>
      <Layout>
        <Component {...pageProps} />
      </Layout>
    </ChakraProvider>
  );
}
