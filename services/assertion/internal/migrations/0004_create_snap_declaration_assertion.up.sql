-- TODO: this needs to change to support all the objects that are in the snapdeclartion assertion
-- currently just a JSON blob for the plugs and slots

-- uuid-ossp or pgcrypto for gen_random_uuid():
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

------------------------------------------------------------------------------
-- master assertion record
------------------------------------------------------------------------------
CREATE TABLE snap_declaration_assertion (
    id                UUID        NOT NULL  PRIMARY KEY  DEFAULT uuid_generate_v4(),
    authority_id      TEXT        NOT NULL,
    sign_key_sha3_384 TEXT        NOT NULL,
    snap_id           TEXT        NOT NULL,
    snap_name         TEXT        NOT NULL,
    publisher_id      TEXT        NOT NULL,
    revision          INTEGER     NOT NULL,
    series            TEXT     NOT NULL,
    timestamp         TIMESTAMPTZ NOT NULL,
    refresh_control   TEXT[]      NOT NULL  DEFAULT '{}',
    
    plugs           JSONB       NOT NULL  DEFAULT '{}'::JSONB,
    slots           JSONB       NOT NULL  DEFAULT '{}'::JSONB,

    signature         TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL  DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL  DEFAULT now()
);

-- ensure only one declaration per (snap_id, revision)
CREATE UNIQUE INDEX ux_sda_snapid_revision
  ON snap_declaration_assertion (snap_id, revision);

CREATE TABLE alias (
    assertion_id UUID      NOT NULL
      REFERENCES snap_declaration_assertion(id)
      ON DELETE CASCADE,
    name          TEXT     NOT NULL,
    target        TEXT     NOT NULL,
    PRIMARY KEY (assertion_id, name)
);
