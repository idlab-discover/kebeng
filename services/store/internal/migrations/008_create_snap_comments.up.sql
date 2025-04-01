CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.comment
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    entry_id uuid,
    author_id uuid,
    reason text COLLATE pg_catalog."default",
    comment text COLLATE pg_catalog."default",
    CONSTRAINT comment_pkey PRIMARY KEY (id),
    CONSTRAINT fk_comment_entry_id FOREIGN KEY (entry_id)
        REFERENCES public.entry (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)
