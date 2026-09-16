-- Invalid legacy roles abort instead of silently changing authority.
ALTER TABLE team_memberships ADD CONSTRAINT team_memberships_role_allowed
  CHECK (role IN ('owner', 'admin', 'member'));
CREATE UNIQUE INDEX team_memberships_one_owner ON team_memberships(team_id) WHERE role = 'owner';
CREATE INDEX teams_invitation_lookup ON teams(upper(btrim(join_code))) WHERE join_mode = 'invitation';
