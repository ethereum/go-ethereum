import { DocSearch } from '@docsearch/react';

import '@docsearch/css';

export const Search: React.FC = () => {
  const appId = process.env.NEXT_PUBLIC_ALGOLIA_APP_ID || '';
  const apiKey = process.env.NEXT_PUBLIC_ALGOLIA_SEARCH_API_KEY || '';
  const indexName = process.env.NEXT_PUBLIC_ALGOLIA_BASE_SEARCH_INDEX_NAME || '';

  // TODO: Replace Algolia test keys with above env vars when ready
  return (
    <DocSearch appId={appId} apiKey={apiKey} indexName={indexName} />
  );
};
