export interface RankingMember {
  user_id: number;
  name: string;
  email: string;
}

export function matchRankingRoster(text: string, members: RankingMember[]) {
  const names = text
    .replace(/^\uFEFF/, "")
    .split(/[\r\n,，;；\t]+/)
    .map((name) => name.trim().replace(/^"(.*)"$/, "$1"))
    .filter(Boolean);
  if (["用户名", "username", "name"].includes(names[0]?.toLowerCase()))
    names.shift();
  const matched: number[] = [],
    missing: string[] = [],
    ambiguous: string[] = [],
    duplicates: string[] = [];
  const seen = new Set<string>();
  for (const name of names) {
    if (seen.has(name)) {
      duplicates.push(name);
      continue;
    }
    seen.add(name);
    const users = members.filter((member) => member.name === name);
    if (!users.length) missing.push(name);
    else if (users.length > 1) ambiguous.push(name);
    else matched.push(users[0].user_id);
  }
  return { matched, missing, ambiguous, duplicates };
}
