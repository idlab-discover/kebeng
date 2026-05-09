CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.snap_deltas
(
	id uuid NOT NULL DEFAULT uuid_generate_v4(),
	created_at timestamp with time zone DEFAULT now(),
	source_revision_id uuid NOT NULL REFERENCES revision(id),
	target_revision_id uuid NOT NULL REFERENCES revision(id),
	minio_file_path TEXT NOT NULL COLLATE pg_catalog."default",
	format TEXT NOT NULL,
	size bigint,
	sha3_384_encoded text COLLATE pg_catalog."default",
    	CONSTRAINT snap_deltas_pkey PRIMARY KEY (id),
	UNIQUE (source_revision_id, target_revision_id, format)
);

CREATE INDEX idx_snap_deltas_source ON public.snap_deltas(source_revision_id);
CREATE INDEX idx_snap_deltas_target ON public.snap_deltas(target_revision_id);
