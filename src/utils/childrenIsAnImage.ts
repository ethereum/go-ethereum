export const childrenIsAnImage = (children: any) => {
  return typeof children[0] === 'object' && children[0].props.hasOwnProperty('src');
};
