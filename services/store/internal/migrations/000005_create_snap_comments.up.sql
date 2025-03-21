CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.snap_comments
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    snap_entry_id uuid,
    author_id uuid,
    reason text COLLATE pg_catalog."default",
    comment text COLLATE pg_catalog."default",
    CONSTRAINT snap_comments_pkey PRIMARY KEY (id),
    CONSTRAINT fk_snap_comments_snap_entry_id FOREIGN KEY (snap_entry_id)
        REFERENCES public.snap_entries (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)
