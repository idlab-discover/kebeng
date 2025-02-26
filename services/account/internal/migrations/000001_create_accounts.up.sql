-- check if manager role exists, if not create
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager') THEN
       CREATE ROLE manager LOGIN;
    END IF;
END
$$;

-- Enable the uuid-ossp extension for uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create sequence public.accounts_id_seq;

CREATE TABLE IF NOT EXISTS public.accounts
(
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    display_name text COLLATE pg_catalog."default",
    username text COLLATE pg_catalog."default",
    email text COLLATE pg_catalog."default",
    password text COLLATE pg_catalog."default",
    CONSTRAINT accounts_pkey PRIMARY KEY (id),
    CONSTRAINT accounts_account_id_key UNIQUE (id),
    CONSTRAINT accounts_display_name_key UNIQUE (display_name),
    CONSTRAINT accounts_username_key UNIQUE (username)
)

TABLESPACE pg_default;

ALTER TABLE public.accounts
    OWNER to manager;
-- Index: idx_accounts_deleted_at

-- DROP INDEX public.idx_accounts_deleted_at;

CREATE INDEX idx_accounts_deleted_at
    ON public.accounts USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
