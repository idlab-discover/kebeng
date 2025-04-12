CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.upload
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    account_id uuid NOT NULL,
    snap_name text COLLATE pg_catalog."default",
    status text COLLATE pg_catalog."default",
    unscanned_file_name text COLLATE pg_catalog."default",
    revision int,
    CONSTRAINT upload_pkey PRIMARY KEY (id),
    entry_id uuid constraint entry_id_fk references entry(id)
);