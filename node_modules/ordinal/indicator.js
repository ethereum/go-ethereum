module.exports = function indicator (i) {
  i = Math.abs(i)
  var cent = i % 100
  if (cent >= 10 && cent <= 20) return 'th'
  var dec = i % 10
  if (dec === 1) return 'st'
  if (dec === 2) return 'nd'
  if (dec === 3) return 'rd'
  return 'th'
}
