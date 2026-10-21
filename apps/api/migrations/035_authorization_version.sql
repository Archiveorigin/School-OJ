ALTER TABLE users ADD COLUMN IF NOT EXISTS perm_version bigint NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version bigint NOT NULL DEFAULT 1;

-- Database-owned invalidation covers all current and future write paths,
-- including author approval, account removal and administrative changes.
CREATE OR REPLACE FUNCTION bump_user_perm_version() RETURNS trigger AS $$
BEGIN
  IF NEW.password_hash IS DISTINCT FROM OLD.password_hash
     OR NEW.account_deleted IS DISTINCT FROM OLD.account_deleted THEN
    NEW.token_version := OLD.token_version + 1;
  ELSE
    NEW.token_version := OLD.token_version;
  END IF;
  IF NEW.role IS DISTINCT FROM OLD.role
     OR NEW.can_author IS DISTINCT FROM OLD.can_author
     OR NEW.account_deleted IS DISTINCT FROM OLD.account_deleted THEN
    NEW.perm_version := OLD.perm_version + 1;
  ELSE
    NEW.perm_version := OLD.perm_version;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS users_perm_version ON users;
CREATE TRIGGER users_perm_version BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION bump_user_perm_version();
