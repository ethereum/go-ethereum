import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [180, 278];

const Icon = createIcon({
  displayName: 'GlyphHome',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M90.0002 276.5V207.379L2.76453 157.376L90.0002 276.5Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
      <path d="M90.0001 276.5V207.379L177.236 157.376L90.0001 276.5Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
      <path d="M89.9999 190.325V102.883L1.5 141.27L89.9999 190.325Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
      <path d="M90.0001 190.325V102.883L178.5 141.27L90.0001 190.325Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
      <path d="M1.5 140.901L89.9999 1.5V102.26L1.5 140.901Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
      <path d="M178.5 140.901L90.0001 1.5V102.26L178.5 140.901Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round"/>
    </svg>
  )
});

export const GlyphHome: React.FC<IconProps> = (props) => <Icon h={h} w={w} color='primary' {...props} />;
