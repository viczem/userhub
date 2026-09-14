CREATE TABLE routes
(
    country CHAR(2) NOT NULL PRIMARY KEY,
    host    TEXT    NOT NULL
);


CREATE TYPE account_status AS ENUM ('registration', 'active', 'reserved');


CREATE TABLE accounts
(
    email      BYTEA          NOT NULL PRIMARY KEY, -- HMAC-SHA-256
    email_kid  SMALLINT       NOT NULL,
    route      CHAR(2)                 DEFAULT NULL REFERENCES routes (country) ON DELETE RESTRICT ON UPDATE RESTRICT,
    status     account_status NOT NULL DEFAULT 'registration',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CHECK (octet_length(email) = 32),
    CHECK (email_kid > 0)
);

CREATE INDEX accounts_route_idx ON accounts (route);
