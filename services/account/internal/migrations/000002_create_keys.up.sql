CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create sequence public.keys_id_seq;

CREATE TABLE IF NOT EXISTS public.keys
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    until timestamp with time zone,
    name text COLLATE pg_catalog."default" NOT NULL,
    sha3384 text COLLATE pg_catalog."default" NOT NULL,
    encoded_public_key text COLLATE pg_catalog."default" NOT NULL,
    account_id uuid,
    CONSTRAINT keys_pkey PRIMARY KEY (id),
    CONSTRAINT keys_sha3384_key UNIQUE (sha3384),
    CONSTRAINT fk_accounts_keys FOREIGN KEY (account_id)
        REFERENCES public.accounts (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE public.keys
    OWNER to manager;
-- Index: idx_keys_deleted_at

-- DROP INDEX public.idx_keys_deleted_at;

CREATE INDEX idx_keys_deleted_at
    ON public.keys USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
