import fs from 'fs';

export const getFileList = (dirName: string) => {
  let files: string[] = [];
  const items = fs.readdirSync(dirName, { withFileTypes: true });

  for (const item of items) {
    if (item.isDirectory()) {
      files = [...files, ...getFileList(`${dirName}/${item.name}`)];
    } else {
      files.push(`/${dirName}/${item.name}`);
    }
  }

  return files.map(file => file.replace('.md', '')).map(file => file.replace('/index', ''));
};
