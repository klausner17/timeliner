-- +goose Up
CREATE TABLE users (
    id            text PRIMARY KEY,
    handle        text NOT NULL UNIQUE,
    name          text NOT NULL,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    bio           text NOT NULL DEFAULT '',
    tint          text NOT NULL DEFAULT 'neutral',
    created_at    timestamptz NOT NULL
);

CREATE TABLE sessions (
    token_hash text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE timelines (
    id            text PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    title         text NOT NULL,
    description   text NOT NULL DEFAULT '',
    category      text NOT NULL,
    color         text NOT NULL,
    owner_id      text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    visibility    text NOT NULL,
    contributions text NOT NULL,
    draft         boolean NOT NULL DEFAULT false,
    eras          jsonb NOT NULL DEFAULT '[]',
    tags          text[] NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL
);
CREATE INDEX timelines_owner_idx ON timelines (owner_id);
CREATE INDEX timelines_tags_idx ON timelines USING gin (tags);

CREATE TABLE events (
    id          text PRIMARY KEY,
    timeline_id text NOT NULL REFERENCES timelines (id) ON DELETE CASCADE,
    year        integer NOT NULL,
    end_year    integer,
    date        text,
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    source_url  text,
    era         text,
    author_id   text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL
);
CREATE INDEX events_timeline_year_idx ON events (timeline_id, year);
CREATE INDEX events_author_idx ON events (author_id);

CREATE TABLE suggestions (
    id               text PRIMARY KEY,
    timeline_id      text NOT NULL REFERENCES timelines (id) ON DELETE CASCADE,
    kind             text NOT NULL,
    status           text NOT NULL,
    author_id        text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- sem FK: a sugestão continua existindo se o evento for apagado
    event_id         text,
    proposed         jsonb NOT NULL,
    note             text,
    feedback         text,
    applied_event_id text,
    previous         jsonb,
    created_at       timestamptz NOT NULL,
    reviewed_at      timestamptz
);
CREATE INDEX suggestions_timeline_idx ON suggestions (timeline_id, status);
CREATE INDEX suggestions_author_idx ON suggestions (author_id);

CREATE TABLE timeline_follows (
    user_id     text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    timeline_id text NOT NULL REFERENCES timelines (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    id          bigserial,
    PRIMARY KEY (user_id, timeline_id)
);
CREATE INDEX timeline_follows_timeline_idx ON timeline_follows (timeline_id);

CREATE TABLE user_follows (
    follower_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followed_id text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    id          bigserial,
    PRIMARY KEY (follower_id, followed_id)
);
CREATE INDEX user_follows_followed_idx ON user_follows (followed_id);

CREATE TABLE saved_comparisons (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slugs      text[] NOT NULL,
    start_year integer NOT NULL,
    end_year   integer NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX saved_comparisons_user_idx ON saved_comparisons (user_id, created_at DESC);

-- +goose Down
DROP TABLE saved_comparisons, user_follows, timeline_follows, suggestions, events, sessions, timelines, users;
