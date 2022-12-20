import { NavLink } from '../types';

interface Props {
  to?: string;
  items?: NavLink[];
  pathCheck: string;
}
export const checkNavLinks = ({ to, items, pathCheck }: Props): boolean => {
  let tracker = false;

  if (to === pathCheck) {
    tracker = true;
  }

  items?.forEach(({ to, items }) => {
    if (checkNavLinks({ to, items, pathCheck })) {
      tracker = true;
    }
  });

  return tracker;
};
