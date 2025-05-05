CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE snap_build_assertion (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    authority_id        TEXT NOT NULL,
    sign_key_sha3_384   TEXT NOT NULL,
    snap_id             UUID NOT NULL,
    account_id          UUID NOT NULL,
    grade               TEXT NOT NULL CHECK (grade IN ('stable', 'devel')),
    snap_sha3_384       TEXT NOT NULL,
    snap_size           BIGINT NOT NULL CHECK (snap_size > 0),
    signature           TEXT NOT NULL,
    timestamp           TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT now()
);