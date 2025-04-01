CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.revision
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    snap_name text COLLATE pg_catalog."default",
    build_assertion_filename text COLLATE pg_catalog."default",
    sha3_384 text COLLATE pg_catalog."default",
    sha3_384_encoded text COLLATE pg_catalog."default",
    size bigint,
    sequence_number bigint,
    architectures TEXT[], -- TODO: check if ok like this?
    status TEXT COLLATE pg_catalog."default",
    version TEXT COLLATE pg_catalog."default",

    CONSTRAINT revision_pkey PRIMARY KEY (id),

    entry_id uuid constraint fk_revision_entry
        references entry,
    snap_track_id uuid constraint fk_revision_track
        references track,
    snap_channel_id uuid constraint fk_revision_channel
        references channel
)

TABLESPACE pg_default;

ALTER TABLE public.revision
    OWNER to manager;

-- Index: idx_revision_deleted_at

-- DROP INDEX public.idx_revision_deleted_at;

CREATE INDEX idx_revision_deleted_at
    ON public.revision USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
