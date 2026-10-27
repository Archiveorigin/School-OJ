import { describe, expect, it } from 'vitest';
import { matchRankingRoster } from '../src/features/teams/rankingRoster';
const members = [{ user_id: 1, name: '甲', email: 'a@test' }, { user_id: 2, name: '乙', email: 'b@test' }, { user_id: 3, name: '同名', email: 'c@test' }, { user_id: 4, name: '同名', email: 'd@test' }];
describe('ranking roster import', () => {
  it('matches exact usernames, handles BOM/header/CSV and deduplicates', () => {
    expect(matchRankingRoster('\uFEFF用户名\r\n"甲",乙\n甲', members)).toEqual({ matched: [1,2], missing: [], ambiguous: [], duplicates: ['甲'] });
  });
  it('reports unknown and ambiguous names without selecting the wrong person', () => {
    expect(matchRankingRoster('未知\n同名\n甲', members)).toEqual({ matched: [1], missing: ['未知'], ambiguous: ['同名'], duplicates: [] });
  });
  it('does not split usernames containing spaces or match partial names', () => {
    expect(matchRankingRoster('Alice Smith\nAlice', [{ user_id: 5, name: 'Alice Smith', email: '' }])).toMatchObject({ matched: [5], missing: ['Alice'] });
  });
});
