CREATE SCHEMA IF NOT EXISTS social;
CREATE SCHEMA IF NOT EXISTS post;

CREATE TABLE IF NOT EXISTS social.users (
    id             bigint PRIMARY KEY,
    handle         text NOT NULL UNIQUE,
    display_name   text NOT NULL,
    follower_count integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS social.follows (
    follower_id bigint NOT NULL REFERENCES social.users (id) ON DELETE CASCADE,
    followee_id bigint NOT NULL REFERENCES social.users (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);

CREATE INDEX IF NOT EXISTS follows_followee_idx
    ON social.follows (followee_id, follower_id);

CREATE TABLE IF NOT EXISTS post.posts (
    id         bigint PRIMARY KEY,
    author_id  bigint NOT NULL,
    body       text NOT NULL,
    media      jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL,
    deleted    boolean NOT NULL DEFAULT false,
    like_count bigint NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS posts_author_idx
    ON post.posts (author_id, id DESC);

CREATE TABLE IF NOT EXISTS post.likes (
    user_id    bigint NOT NULL,
    post_id    bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS likes_post_idx
    ON post.likes (post_id);

CREATE TABLE IF NOT EXISTS post.outbox (
    id           bigserial PRIMARY KEY,
    event_type   text NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS outbox_unpublished_idx
    ON post.outbox (id)
    WHERE published_at IS NULL;
