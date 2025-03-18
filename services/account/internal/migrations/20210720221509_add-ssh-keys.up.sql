CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create sequence public.ssh_keys_id_seq;

CREATE TABLE IF NOT EXISTS public.ssh_keys
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    public_key_string text COLLATE pg_catalog."default" NOT NULL,
    account_id uuid,
    CONSTRAINT ssh_keys_pkey PRIMARY KEY (id),
    CONSTRAINT fk_accounts_ssh_keys FOREIGN KEY (account_id)
        REFERENCES public.accounts (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

    TABLESPACE pg_default;

ALTER TABLE public.ssh_keys
    OWNER to manager;

CREATE INDEX idx_ssh_keys_deleted_at
    ON public.ssh_keys USING btree
        (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
