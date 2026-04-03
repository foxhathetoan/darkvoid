DROP TRIGGER IF EXISTS trg_follow_counts ON usr.follows;
DROP FUNCTION IF EXISTS usr.update_follow_counts();
ALTER TABLE usr.users
    DROP COLUMN IF EXISTS follower_count,
    DROP COLUMN IF EXISTS following_count;
