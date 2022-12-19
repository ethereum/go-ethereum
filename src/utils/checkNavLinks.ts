import { NavLink } from "../types";

interface Props {
  to?: string
  items?: NavLink[]
  pathCheck: string
  tracker?: boolean
}
export const checkNavLinks = ({ to, items, pathCheck, tracker = false }: Props): boolean => {
  if (to === pathCheck) {
    tracker = true
  }

  items?.forEach(({to, items}) => {    
    if (checkNavLinks({to, items, pathCheck})){
      tracker = true
    }
  });

  return tracker
}