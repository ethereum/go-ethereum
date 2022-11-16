import { IconProps } from '@chakra-ui/react';
import { createIcon } from '@chakra-ui/icons';

const [w, h] = [25, 30];

const Icon = createIcon({
  displayName: 'MacosLogo',
  viewBox: `0 0 ${w} ${h}`,
  path: (
    <svg width={w} height={h} fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M25.0003 22.0095C24.0178 24.8596 21.0764 29.906 18.0463 29.961C16.0363 29.9997 15.39 28.7697 13.0924 28.7697C10.7961 28.7697 10.0773 29.9235 8.17728 29.9985C4.96217 30.1222 -0.000488281 22.7145 -0.000488281 16.2543C-0.000488281 10.3203 4.13465 7.37899 7.74726 7.32524C9.68483 7.29024 11.5149 8.63153 12.6962 8.63153C13.8825 8.63153 16.105 7.01898 18.4414 7.25523C19.4189 7.29649 22.1652 7.649 23.9278 10.2266C19.2514 13.2792 19.9802 19.6631 25.0003 22.0095ZM18.4726 0C14.94 0.142505 12.0574 3.84887 12.4599 6.91397C15.725 7.16773 18.8576 3.50761 18.4726 0Z" fill="currentColor"/>
    </svg>
  )
});

export const MacosLogo: React.FC<IconProps> = (props) => <Icon h={h} w={w} color='bg' {...props} />; // #F0F2E2
