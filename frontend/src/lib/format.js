export function number(value = 0) {
  return new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(
    value
  );
}
export function timeAgo(value) {
  if (!value || new Date(value).getUTCFullYear() < 1970) return '—';
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000));
  for (const [unit, size] of [
    ['year', 31536000],
    ['month', 2592000],
    ['day', 86400],
    ['hour', 3600],
    ['minute', 60],
    ['second', 1]
  ]) {
    if (seconds >= size || size === 1) {
      const count = Math.floor(seconds / size);
      return `${count} ${unit}${count === 1 ? '' : 's'}`;
    }
  }
}
export const periods = [
  ['1h', 'Last hour'],
  ['24h', 'Last 24 hours'],
  ['7d', 'Last 7 days'],
  ['14d', 'Last 14 days'],
  ['30d', 'Last 30 days'],
  ['90d', 'Last 90 days']
];
