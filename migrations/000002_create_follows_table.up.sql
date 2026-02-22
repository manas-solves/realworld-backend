CREATE TABLE follows
(
    follower_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followed_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (follower_id, followed_id)
);
