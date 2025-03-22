CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.snap_revisions
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    snap_name text COLLATE pg_catalog."default",
    snap_entry_id uuid,   
    build_assertion_filename text COLLATE pg_catalog."default",
    sha3_384 text COLLATE pg_catalog."default",
    sha3_384_encoded text COLLATE pg_catalog."default",
    size bigint,
    sequence_number bigint,
    architectures TEXT[], -- TODO: check if ok like this?
    status TEXT COLLATE pg_catalog."default",
    version TEXT COLLATE pg_catalog."default",

    CONSTRAINT snap_revisions_pkey PRIMARY KEY (id),
    CONSTRAINT fk_snap_entries_revisions FOREIGN KEY (snap_entry_id)
        REFERENCES public.snap_entries (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION 
)

TABLESPACE pg_default;

ALTER TABLE public.snap_revisions
    OWNER to manager;

-- Index: idx_snap_revisions_deleted_at

-- DROP INDEX public.idx_snap_revisions_deleted_at;

CREATE INDEX idx_snap_revisions_deleted_at
    ON public.snap_revisions USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
