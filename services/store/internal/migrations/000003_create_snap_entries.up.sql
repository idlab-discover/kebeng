-- check if manager role exists, if not create
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager') THEN
       CREATE ROLE manager LOGIN;
    END IF;
END
$$;


CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create sequence public.snap_entries_id_seq;

CREATE TABLE IF NOT EXISTS public.snap_entries
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text COLLATE pg_catalog."default",
    latest_revision_id bigint,
    type text COLLATE pg_catalog."default",
    confinement text COLLATE pg_catalog."default",
    account_id uuid,
    CONSTRAINT snap_entries_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE public.snap_entries
    OWNER to manager;

-- Index: idx_snap_entries_deleted_at

-- DROP INDEX public.idx_snap_entries_deleted_at;

CREATE INDEX idx_snap_entries_deleted_at
    ON public.snap_entries USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
