import { Html, Head, Main, NextScript } from 'next/document';

export default function Document() {
  return (
    <Html lang='en'>
      <Head>
        {/* fonts are being loaded here to enable optimization (https://nextjs.org/docs/basic-features/font-optimization) */}
        {/* JetBrains Mono */}
        <link rel='preconnect' href='https://fonts.googleapis.com' />
        <link rel='preconnect' href='https://fonts.gstatic.com' crossOrigin='true' />
        <link
          href='https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap'
          rel='stylesheet'
        />

        {/* Inter */}
        <link
          href='https://fonts.googleapis.com/css2?family=Inter:wght@400;700&display=swap'
          rel='stylesheet'
        ></link>
      </Head>

      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  );
}
