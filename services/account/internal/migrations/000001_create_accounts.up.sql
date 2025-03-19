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
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    display_name TEXT COLLATE pg_catalog."default" UNIQUE NOT NULL CHECK (display_name <> ''),
    username TEXT COLLATE pg_catalog."default" UNIQUE NOT NULL CHECK (username <> ''),
    email TEXT COLLATE pg_catalog."default" UNIQUE NOT NULL CHECK (email <> ''),
    password_hash TEXT COLLATE pg_catalog."default" NOT NULL CHECK (password_hash <> ''),
    validation TEXT COLLATE pg_catalog."default",
    CONSTRAINT accounts_pkey PRIMARY KEY (id),
    CONSTRAINT accounts_display_name_key UNIQUE (display_name),
    CONSTRAINT accounts_username_key UNIQUE (username),
    CONSTRAINT accounts_email_key UNIQUE (email)
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
