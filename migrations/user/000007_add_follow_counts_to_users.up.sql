ALTER TABLE usr.users
    ADD COLUMN follower_count  BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN following_count BIGINT NOT NULL DEFAULT 0;

-- Backfill existing counts
UPDATE usr.users u
SET
    follower_count  = (SELECT COUNT(*) FROM usr.follows f WHERE f.followee_id = u.id),
    following_count = (SELECT COUNT(*) FROM usr.follows f WHERE f.follower_id = u.id);

-- Trigger function: maintain counters on INSERT/DELETE from usr.follows
CREATE OR REPLACE FUNCTION usr.update_follow_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE usr.users SET follower_count  = follower_count  + 1 WHERE id = NEW.followee_id;
        UPDATE usr.users SET following_count = following_count + 1 WHERE id = NEW.follower_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE usr.users SET follower_count  = GREATEST(follower_count  - 1, 0) WHERE id = OLD.followee_id;
        UPDATE usr.users SET following_count = GREATEST(following_count - 1, 0) WHERE id = OLD.follower_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_follow_counts
AFTER INSERT OR DELETE ON usr.follows
FOR EACH ROW EXECUTE FUNCTION usr.update_follow_counts();
