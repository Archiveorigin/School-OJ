export function parseTeamJoinSlug(value: string) {
  let slug = value.trim().toLowerCase()
  if (slug.includes('/')) {
    try { slug = new URL(slug, 'https://team.local').pathname.match(/^\/teams\/([a-z][a-z0-9-]{1,29})(?:\/join)?\/?$/)?.[1] || '' }
    catch { return '' }
  }
  return /^[a-z][a-z0-9-]{1,29}$/.test(slug) ? slug : ''
}
