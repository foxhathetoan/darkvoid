CREATE TABLE usr.follows (
    follower_id UUID        NOT NULL,
    followee_id UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT chk_no_self_follow CHECK (follower_id <> followee_id)
);

CREATE INDEX idx_usr_follows_follower ON usr.follows(follower_id);
CREATE INDEX idx_usr_follows_followee ON usr.follows(followee_id);
