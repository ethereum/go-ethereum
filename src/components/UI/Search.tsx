import { DocSearch } from '@docsearch/react';

import '@docsearch/css';

export const Search: React.FC = () => {
  // TODO: Replace Algolia test keys with '' when API key available for this repo (place in .env)
  const appId = process.env.NEXT_PUBLIC_ALGOLIA_APP_ID || 'R2IYF7ETH7'
  const apiKey = process.env.NEXT_PUBLIC_ALGOLIA_SEARCH_API_KEY || '599cec31baffa4868cae4e79f180729b'
  const indexName = process.env.NEXT_PUBLIC_ALGOLIA_BASE_SEARCH_INDEX_NAME || 'docsearch'
  return (
    <DocSearch
      appId={appId}
      apiKey={apiKey}
      indexName={indexName} />
  );
}
