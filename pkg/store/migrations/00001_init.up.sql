-- +goose Up
CREATE TABLE IF NOT EXISTS identities(
    id TEXT PRIMARY KEY UNIQUE,
    passw TEXT
);

CREATE TABLE IF NOT EXISTS resources(
    id SERIAL PRIMARY KEY UNIQUE,
    resource INTEGER,
    type INTEGER,
    owner TEXT,
    meta TEXT
);

CREATE TABLE IF NOT EXISTS pieces(
    id SERIAL PRIMARY KEY UNIQUE,
    content BYTEA,
    salt BYTEA,
    iv BYTEA
);

CREATE TABLE IF NOT EXISTS blobs(
    id SERIAL PRIMARY KEY UNIQUE,
    location TEXT,
    salt BYTEA,
    iv BYTEA
);


INSERT INTO
    identities (id, passw)
        VALUES
            ('stas', '$2a$10$k4/iXqhXQg/mK/fsDXbF5Ocq50yPzkaw4l4Elg37A38fYmtw7oxAm'),
            ('nata', '$2a$10$7ixg.hUXcUF4YTHZfgrU.ePgOhvAZhu5sIaOa4TTTwgIfxIhVnMry');

-- +goose Down
DROP TABLE identities;
DROP TABLE resources;
DROP TABLE pieces;
DROP TABLE blobs;
