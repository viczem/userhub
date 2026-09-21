CREATE TABLE runtime
(
    key   TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    CHECK (length(btrim(key)) > 0),
    CHECK (jsonb_typeof(value) = 'object')
);


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


CREATE TABLE locales
(
    code        VARCHAR(35) PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);


CREATE TABLE locale_aliases
(
    alias  VARCHAR(35),
    locale VARCHAR(35) NOT NULL
        REFERENCES locales (code)
            ON DELETE RESTRICT
            ON UPDATE RESTRICT,

    UNIQUE NULLS NOT DISTINCT (alias)
);

CREATE INDEX locale_aliases_locale_idx ON locale_aliases (locale);


CREATE TABLE accounts
(
    id          UUID PRIMARY KEY,
    locale      VARCHAR(35) NOT NULL REFERENCES locales (code) ON DELETE RESTRICT ON UPDATE RESTRICT,
    email       BYTEA       NOT NULL, -- HMAC-SHA-256
    email_kid   SMALLINT    NOT NULL,
    password    TEXT DEFAULT NULL, -- Argon2id (optional)
    profile     BYTEA       NOT NULL, -- AES-256-GCM (json data: email, etc.)
    profile_kid SMALLINT    NOT NULL,
    UNIQUE (email),
    CHECK (octet_length(email) = 32),
    CHECK (email_kid > 0),
    CHECK (profile_kid > 0)
);


CREATE TABLE sessions
(
    id             UUID PRIMARY KEY,
    account_id     UUID        NOT NULL
        REFERENCES accounts (id)
            ON DELETE CASCADE
            ON UPDATE RESTRICT,
    jti            UUID        NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    prev_jti       UUID,
    prev_jti_until TIMESTAMPTZ,
    user_agent     TEXT,
    browser        TEXT,
    os             TEXT,
    device_type    TEXT,
    country_code   VARCHAR(2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ NOT NULL,

    CHECK (expires_at > created_at),
    CHECK (issued_at >= created_at AND issued_at < expires_at),
    CHECK (
        (prev_jti IS NULL AND prev_jti_until IS NULL) OR (
            prev_jti IS NOT NULL
                AND prev_jti_until IS NOT NULL
                AND prev_jti <> jti
                AND prev_jti_until > issued_at
                AND prev_jti_until <= expires_at
        )
    ),
    CHECK (country_code IS NULL OR country_code ~ '^[A-Z]{2}$')
);

CREATE INDEX sessions_account_id_idx ON sessions (account_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);


CREATE TABLE roles
(
    name        VARCHAR(100) PRIMARY KEY,
    description TEXT    NOT NULL DEFAULT '',
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,

    CHECK (name ~ '^[a-z][a-z0-9_]*$')
);

CREATE UNIQUE INDEX roles_single_default_idx ON roles (is_default) WHERE is_default;


CREATE TABLE account_roles
(
    account_id UUID         NOT NULL,
    role       VARCHAR(100) NOT NULL,

    PRIMARY KEY (account_id, role),

    FOREIGN KEY (account_id)
        REFERENCES accounts (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,

    FOREIGN KEY (role)
        REFERENCES roles (name)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT
);

CREATE INDEX account_roles_role_idx ON account_roles (role);


CREATE TABLE account_registrations
(
    email       BYTEA       NOT NULL PRIMARY KEY,                                                                 -- HMAC-SHA-256
    email_kid   SMALLINT    NOT NULL,
    payload     BYTEA       NOT NULL,                                                                             -- AES-256-GCM encrypted JSON; always contains the submitted email
    payload_kid SMALLINT    NOT NULL,
    otp_id      UUID                 DEFAULT NULL REFERENCES otp (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    account_id  UUID                 DEFAULT NULL REFERENCES accounts (id) ON DELETE RESTRICT ON UPDATE RESTRICT, -- Email owner after confirmation; prepopulated for an email change
    locale      VARCHAR(35) NOT NULL REFERENCES locales (code) ON DELETE RESTRICT ON UPDATE RESTRICT,
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

CREATE INDEX account_registrations_account_id_idx ON account_registrations (account_id);


CREATE TABLE email_templates
(
    name      VARCHAR(250) NOT NULL,
    locale    VARCHAR(35)  NOT NULL REFERENCES locales (code) ON DELETE RESTRICT ON UPDATE RESTRICT,
    subject   TEXT         NOT NULL,
    text_body TEXT         NOT NULL,
    html_body TEXT,
    PRIMARY KEY (name, locale)
);

CREATE INDEX email_templates_locale_idx ON email_templates (locale);


CREATE TABLE email_outbox
(
    id          UUID PRIMARY KEY,
    template    VARCHAR(250) NOT NULL,
    locale      VARCHAR(35)  NOT NULL,
    payload     BYTEA        NOT NULL, -- AES-256-GCM encrypted JSON
    payload_kid SMALLINT     NOT NULL,
    claim       INTEGER      NOT NULL DEFAULT 0,
    claim_until TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ  NOT NULL,

    FOREIGN KEY (template, locale)
        REFERENCES email_templates (name, locale)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT,

    CHECK (payload_kid > 0),
    CHECK (claim >= 0)
);

CREATE INDEX email_outbox_available_idx ON email_outbox (claim_until);
CREATE INDEX email_outbox_expiration_idx ON email_outbox (expires_at);
CREATE INDEX email_outbox_template_locale_idx ON email_outbox (template, locale);
