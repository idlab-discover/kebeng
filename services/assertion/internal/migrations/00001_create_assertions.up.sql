CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.assertions
(
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone,
    deleted_at timestamp with time zone,
    assertion TEXT UNIQUE, -- Changed from blob to text to store UTF-8 encoded data
    CONSTRAINT assertions_pkey PRIMARY KEY (id),
    CONSTRAINT assertions_id_key UNIQUE (id),

    snap_entry_id UUID NOT NULL
)
