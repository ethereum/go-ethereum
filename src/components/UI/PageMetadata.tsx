import Head from 'next/head';
import { useRouter } from 'next/router';

import { SITE_NAME, SITE_URL } from '../../constants';

interface Props {
  title: string;
  description: string;
  image?: string;
}

export const PageMetadata: React.FC<Props> = ({ title, description, image }) => {
  const router = useRouter();
  const url = `${SITE_URL}${router.asPath}`;
  const fullTitle = `${title} | ${SITE_NAME}`;
  const defaultOgImage = `${SITE_URL}/images/metadata-gopher.png`;
  const ogImage = !image ? defaultOgImage : `${SITE_URL}${image}`;

  return (
    <Head>
      <title>{fullTitle}</title>
      <meta name='title' content={fullTitle} />
      <meta name='description' content={description} />
      <meta name='application-name' content={SITE_NAME} />
      <meta name='image' content={ogImage} />
      {/* OpenGraph */}
      <meta property='og:title' content={fullTitle} />
      <meta property='og:description' content={description} />
      <meta property='og:type' content='website' />
      <meta property='og:site_name' content={SITE_NAME} />
      <meta property='og:url' content={url} />
      <meta property='og:image' content={ogImage} />
      <meta property='og:image:url' content={ogImage} />
      <meta property='og:image:secure_url' content={ogImage} />
      <meta property='og:image:alt' content={SITE_NAME} />
      <meta property='og:image:type' content='image/png' />
      {/* Twitter */}
      <meta name='twitter:card' content='summary_large_image' />
      <meta property='twitter:url' content={url} />
      <meta name='twitter:creator' content='@go_ethereum' />
      <meta name='twitter:site' content='@go_ethereum' />
      <meta name='twitter:title' content={fullTitle} />
      <meta name='twitter:description' content={description} />
      {/* patch to force a cache invalidation of twitter's card bot */}
      <meta name='twitter:image' content={`${ogImage}/#`} />
      {/* viewport */}
      <meta name='viewport' content='width=device-width, initial-scale=1' />
      {/* favicon */}
      <link rel='icon' type='image/x-icon' href='/favicon.ico' />
    </Head>
  );
};
