import { Flex, FlexProps } from '@chakra-ui/react'
import { DocSearch } from '@docsearch/react';

import '@docsearch/css';

export const Search: React.FC<FlexProps> = (props) => (
  <Flex
    css={`
      svg.DocSearch-Search-Icon path {
        color: var(--chakra-colors-primary);
      }
      .DocSearch-Button {
        border-radius: 0;
        height: 100%;
        background: none;
        margin: 0;
        padding: 1rem;
        width: 200px;
      }
      .DocSearch-Button:hover {
        background: none;
      }
      .DocSearch-Button-Container {
        flex-direction: row-reverse;
      }
      .DocSearch-Button-Keys {
        display: none;
      }
      .DocSearch-Button-Placeholder {
        text-transform: lowercase;
        font-style: italic;
        color: var(--chakra-colors-primary);
        font-weight: 400;
        width: 100%;
        flex: 1;
      }
    `}
    {...props}
  >
    <DocSearch
      appId="R2IYF7ETH7"
      apiKey="599cec31baffa4868cae4e79f180729b"
      indexName="docsearch" />
  </Flex>
)
