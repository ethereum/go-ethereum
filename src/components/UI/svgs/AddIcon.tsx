import { createIcon } from '@chakra-ui/icons';

export const AddIcon = createIcon({
  displayName: 'AddIcon',
  viewBox: '0 0 24 24',
  path: (
    <g fill='currentColor'>
      <path d='M2 11h20v2H2z' />
      <path d='M11 2h2v20h-2z' />
    </g>
  ),
  defaultProps: {
    color: 'primary'
  }
});
