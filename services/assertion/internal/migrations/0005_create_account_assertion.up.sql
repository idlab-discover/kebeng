CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS account_assertion (
    id                UUID            NOT NULL  PRIMARY KEY  DEFAULT uuid_generate_v4(),
    authority_id      TEXT            NOT NULL,
    display_name      TEXT            NOT NULL,
    username          TEXT            NOT NULL,
    validation        TEXT            NULL,
    account_id        UUID            NOT NULL,
    revision          INTEGER         NOT NULL,
    timestamp         TIMESTAMPTZ     NOT NULL,
    sign_key_sha3_384 TEXT            NOT NULL,
    signature         TEXT            NOT NULL
);

-- index for faster lookups by account_id
CREATE INDEX IF NOT EXISTS idx_account_assertion_account_id
  ON account_assertion(account_id);

-- index on revision to allow finding latest
CREATE INDEX IF NOT EXISTS idx_account_assertion_account_revision
  ON account_assertion(account_id, revision DESC);

CREATE UNIQUE INDEX uq_account_assertion_account_rev
  ON account_assertion (account_id, revision);
