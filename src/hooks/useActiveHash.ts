import { useState, useEffect } from 'react';

/**
 * A hook to determine which section of the page is currently in the viewport.
 * @param {*} itemIds Array of document ids to observe
 * @param {*} rootMargin
 * @returns id of the element currently in viewport
 */
export const useActiveHash = (itemIds: Array<string>, rootMargin = `0% 0% -80% 0%`): string => {
  const [activeHash, setActiveHash] = useState(``);

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            setActiveHash(`${entry.target.id}`);
          }
        });
      },
      { rootMargin }
    );

    itemIds?.forEach(id => {
      const element = document.getElementById(id);
      if (element !== null) {
        observer.observe(element);
      }
    });

    return () => {
      itemIds?.forEach(id => {
        const element = document.getElementById(id);
        if (element !== null) {
          observer.unobserve(element);
        }
      });
    };
  }, [itemIds, rootMargin]);

  return activeHash;
};
