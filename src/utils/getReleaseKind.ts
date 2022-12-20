export const getReleaseKind = (filename: string) => {
  const os = filename.includes('alltools') ? filename.split('-')[2] : filename.split('-')[1];

  if (os == 'android' || os == 'ios') {
    return 'Library';
  }

  if (os == 'windows') {
    if (filename.endsWith('.exe')) {
      return 'Installer';
    } else {
      return 'Library';
    }
  }

  return 'Archive';
};
