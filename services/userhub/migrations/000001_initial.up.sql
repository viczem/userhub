CREATE TABLE otp
(
    id         UUID PRIMARY KEY,     -- uuid v4 (random)
    secret     BYTEA       NOT NULL, -- HMAC-SHA-256(SECRET_KEY, otp.id || ":" || otp)
    secret_kid SMALLINT    NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    attempts   INTEGER     NOT NULL DEFAULT 0,
    CHECK (octet_length(secret) = 32),
    CHECK (secret_kid > 0),
    CHECK (attempts >= 0),
    CHECK (expires_at > created_at AND updated_at >= created_at AND updated_at <= expires_at)
);

CREATE INDEX otp_expired_at_idx ON otp (expires_at);


CREATE TABLE accounts
(
    id          UUID PRIMARY KEY,
    email       BYTEA    NOT NULL, -- HMAC-SHA-256
    email_kid   SMALLINT NOT NULL,
    password    TEXT DEFAULT NULL, -- Argon2id (optional)
    profile     BYTEA    NOT NULL, -- AES-256-GCM (json data: email, etc.)
    profile_kid SMALLINT NOT NULL,
    UNIQUE (email),
    CHECK (octet_length(email) = 32),
    CHECK (email_kid > 0),
    CHECK (profile_kid > 0)
);


CREATE TABLE account_registrations
(
    email       BYTEA       NOT NULL PRIMARY KEY,                                                                 -- HMAC-SHA-256
    email_kid   SMALLINT    NOT NULL,
    payload     BYTEA       NOT NULL,                                                                             -- AES-256-GCM encrypted JSON; always contains the submitted email
    payload_kid SMALLINT    NOT NULL,
    otp_id      UUID                 DEFAULT NULL REFERENCES otp (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    account_id  UUID                 DEFAULT NULL REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE RESTRICT, -- Email owner after confirmation; prepopulated for an email change
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (otp_id),
    CHECK (octet_length(email) = 32),
    CHECK (email_kid > 0),
    CHECK (payload_kid > 0),
    CHECK (
        otp_id IS NOT NULL                                                                                        -- Pending verification
            OR (otp_id IS NULL AND account_id IS NOT NULL)                                                        -- Confirmed and permanently reserved email
        )
);

CREATE INDEX account_registrations_account_id_idx
    ON account_registrations (account_id);
