const check = '{#';

export const parseHeadingId = (children: string[]) => {
  if (children[children.length - 1].includes(check)) {
    const temp = children[children.length - 1].split(check);
    const headingId = temp[temp.length - 1].split('}')[0];

    children[children.length - 1] = temp[0];

    return {
      children,
      title: temp[0].replaceAll('#', ''),
      headingId
    };
  }

  return null;
};
