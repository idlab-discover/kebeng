CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.upload
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    up_down_id text COLLATE pg_catalog."default",
    filesize bigint,
    entry_id uuid,
    channels text[],
    CONSTRAINT upload_pkey PRIMARY KEY (id),
    CONSTRAINT fk_entry_uploads FOREIGN KEY (entry_id)
        REFERENCES public.entry (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
) TABLESPACE pg_default;

ALTER TABLE public.upload
    OWNER to manager;