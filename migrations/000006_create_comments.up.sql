CREATE TABLE comments
(
    id         SERIAL PRIMARY KEY,
    body       TEXT      NOT NULL,
    article_id INTEGER   NOT NULL,
    author_id  BIGINT    NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    FOREIGN KEY (article_id) REFERENCES articles (id) ON DELETE CASCADE,
    FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Index for better query performance
CREATE INDEX idx_comments_article_id ON comments (article_id);
CREATE INDEX idx_comments_author_id ON comments (author_id);
CREATE INDEX idx_comments_created_at ON comments (created_at DESC);

