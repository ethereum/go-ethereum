export const getParsedDate = (
  date: string | Date,
  dateOptions: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'UTC'
  } as const
) => {
  return new Date(date).toLocaleDateString('en', dateOptions);
};
